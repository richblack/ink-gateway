-- 多模態向量支援遷移腳本
-- 將向量功能整合到統一的 chunks 表中，支援文字和圖片向量

-- 1. 為 chunks 表新增向量相關欄位
ALTER TABLE chunks 
ADD COLUMN IF NOT EXISTS vector vector(512),  -- 統一使用 512 維向量（CLIP 和 text-embedding-3-small）
ADD COLUMN IF NOT EXISTS vector_type VARCHAR(50) DEFAULT 'text',  -- 'text' 或 'image'
ADD COLUMN IF NOT EXISTS vector_model VARCHAR(100) DEFAULT 'text-embedding-3-small',  -- 模型名稱
ADD COLUMN IF NOT EXISTS vector_metadata JSONB;  -- 向量相關的元資料

-- 2. 建立向量類型索引（分離文字和圖片向量以提升效能）
CREATE INDEX IF NOT EXISTS idx_chunks_text_vectors 
ON chunks USING ivfflat (vector vector_cosine_ops) 
WHERE vector_type = 'text' AND vector IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_chunks_image_vectors 
ON chunks USING ivfflat (vector vector_cosine_ops) 
WHERE vector_type = 'image' AND vector IS NOT NULL;

-- 3. 建立向量類型和模型的一般索引
CREATE INDEX IF NOT EXISTS idx_chunks_vector_type ON chunks(vector_type) WHERE vector IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_chunks_vector_model ON chunks(vector_model) WHERE vector IS NOT NULL;

-- 4. 建立複合索引用於多模態搜尋
CREATE INDEX IF NOT EXISTS idx_chunks_vector_type_model ON chunks(vector_type, vector_model) WHERE vector IS NOT NULL;

-- 5. 更新現有的向量搜尋函數以支援多模態
CREATE OR REPLACE FUNCTION public.match_chunks_multimodal(
    query_embedding vector(512),
    vector_type_filter text DEFAULT 'all',  -- 'text', 'image', 'all'
    match_threshold float DEFAULT 0.7,
    match_count int DEFAULT 10
)
RETURNS TABLE (
    chunk jsonb,
    similarity float
)
LANGUAGE sql
STABLE
AS $$
    SELECT 
        to_jsonb(c.*) as chunk,
        1 - (c.vector <=> query_embedding) as similarity
    FROM chunks c
    WHERE c.vector IS NOT NULL
      AND (vector_type_filter = 'all' OR c.vector_type = vector_type_filter)
      AND 1 - (c.vector <=> query_embedding) > match_threshold
    ORDER BY c.vector <=> query_embedding
    LIMIT match_count;
$$;

-- 6. 建立混合搜尋函數（文字+圖片）
CREATE OR REPLACE FUNCTION public.hybrid_search_chunks(
    text_embedding vector(512),
    image_embedding vector(512) DEFAULT NULL,
    text_weight float DEFAULT 0.7,
    image_weight float DEFAULT 0.3,
    match_threshold float DEFAULT 0.7,
    match_count int DEFAULT 10
)
RETURNS TABLE (
    chunk jsonb,
    similarity float,
    match_type text
)
LANGUAGE sql
STABLE
AS $$
    WITH text_matches AS (
        SELECT 
            c.*,
            (1 - (c.vector <=> text_embedding)) * text_weight as text_sim,
            0.0 as image_sim,
            'text' as match_type
        FROM chunks c
        WHERE c.vector IS NOT NULL 
          AND c.vector_type = 'text'
          AND 1 - (c.vector <=> text_embedding) > match_threshold
    ),
    image_matches AS (
        SELECT 
            c.*,
            0.0 as text_sim,
            CASE 
                WHEN image_embedding IS NOT NULL THEN (1 - (c.vector <=> image_embedding)) * image_weight
                ELSE 0.0
            END as image_sim,
            'image' as match_type
        FROM chunks c
        WHERE c.vector IS NOT NULL 
          AND c.vector_type = 'image'
          AND image_embedding IS NOT NULL
          AND 1 - (c.vector <=> image_embedding) > match_threshold
    ),
    combined_matches AS (
        SELECT *, text_sim + image_sim as total_similarity FROM text_matches
        UNION ALL
        SELECT *, text_sim + image_sim as total_similarity FROM image_matches
    )
    SELECT 
        to_jsonb(cm.*) as chunk,
        cm.total_similarity as similarity,
        cm.match_type
    FROM combined_matches cm
    WHERE cm.total_similarity > match_threshold
    ORDER BY cm.total_similarity DESC
    LIMIT match_count;
$$;

-- 7. 建立圖片去重檢查函數
CREATE OR REPLACE FUNCTION public.find_duplicate_images(
    file_hash text
)
RETURNS TABLE (
    chunk_id uuid,
    storage_url text,
    created_time timestamp with time zone
)
LANGUAGE sql
STABLE
AS $$
    SELECT 
        c.chunk_id,
        c.metadata->>'storage'->>'url' as storage_url,
        c.created_time
    FROM chunks c
    WHERE c.metadata->>'media_type' = 'image'
      AND c.metadata->>'storage'->>'file_hash' = file_hash
    ORDER BY c.created_time ASC;
$$;

-- 8. 建立向量統計檢視
CREATE OR REPLACE VIEW public.vector_statistics AS
SELECT 
    vector_type,
    vector_model,
    COUNT(*) as count,
    MIN(created_time) as first_created,
    MAX(created_time) as last_created
FROM chunks 
WHERE vector IS NOT NULL
GROUP BY vector_type, vector_model
ORDER BY vector_type, vector_model;

-- 9. 新增約束確保資料完整性
ALTER TABLE chunks 
ADD CONSTRAINT check_vector_type 
CHECK (vector_type IN ('text', 'image'));

ALTER TABLE chunks 
ADD CONSTRAINT check_vector_consistency 
CHECK (
    (vector IS NULL AND vector_type IS NULL AND vector_model IS NULL) OR
    (vector IS NOT NULL AND vector_type IS NOT NULL AND vector_model IS NOT NULL)
);

-- 10. 建立觸發器自動更新 last_updated
CREATE OR REPLACE FUNCTION update_chunk_vector_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.vector IS DISTINCT FROM NEW.vector OR 
       OLD.vector_type IS DISTINCT FROM NEW.vector_type OR 
       OLD.vector_model IS DISTINCT FROM NEW.vector_model THEN
        NEW.last_updated = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_chunk_vector_timestamp
    BEFORE UPDATE ON chunks
    FOR EACH ROW 
    EXECUTE FUNCTION update_chunk_vector_timestamp();

-- 11. 新增註解說明
COMMENT ON COLUMN chunks.vector IS '512維向量，支援文字和圖片嵌入';
COMMENT ON COLUMN chunks.vector_type IS '向量類型：text（文字）或 image（圖片）';
COMMENT ON COLUMN chunks.vector_model IS '生成向量的模型名稱，如 text-embedding-3-small 或 clip-vit-b-32';
COMMENT ON COLUMN chunks.vector_metadata IS '向量相關元資料，如置信度、處理參數等';

-- 12. 顯示遷移完成訊息
DO $$
BEGIN
    RAISE NOTICE '✅ 多模態向量支援遷移完成！';
    RAISE NOTICE '📊 新增欄位: vector, vector_type, vector_model, vector_metadata';
    RAISE NOTICE '🔍 建立索引: 文字向量索引、圖片向量索引、複合索引';
    RAISE NOTICE '⚡ 新增函數: match_chunks_multimodal, hybrid_search_chunks, find_duplicate_images';
    RAISE NOTICE '📈 建立檢視: vector_statistics';
    RAISE NOTICE '🛡️ 新增約束: 向量類型檢查、資料一致性檢查';
END $$;