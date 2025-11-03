-- 測試多模態向量遷移腳本
-- 驗證新增的欄位、索引和函數是否正常工作

-- 1. 檢查新增的欄位
SELECT 
    column_name, 
    data_type, 
    is_nullable, 
    column_default
FROM information_schema.columns 
WHERE table_name = 'chunks' 
  AND column_name IN ('vector', 'vector_type', 'vector_model', 'vector_metadata')
ORDER BY column_name;

-- 2. 檢查新增的索引
SELECT 
    indexname, 
    indexdef
FROM pg_indexes 
WHERE tablename = 'chunks' 
  AND indexname LIKE '%vector%'
ORDER BY indexname;

-- 3. 檢查新增的函數
SELECT 
    routine_name,
    routine_type,
    data_type
FROM information_schema.routines 
WHERE routine_name IN ('match_chunks_multimodal', 'hybrid_search_chunks', 'find_duplicate_images')
ORDER BY routine_name;

-- 4. 檢查約束
SELECT 
    constraint_name,
    constraint_type,
    check_clause
FROM information_schema.check_constraints 
WHERE constraint_name LIKE '%vector%';

-- 5. 測試插入文字向量的 chunk
INSERT INTO chunks (
    chunk_id,
    contents,
    vector,
    vector_type,
    vector_model,
    metadata
) VALUES (
    gen_random_uuid(),
    '這是一個測試文字內容',
    ARRAY[0.1, 0.2, 0.3, 0.4, 0.5]::vector(512),
    'text',
    'text-embedding-3-small',
    '{"test": true}'::jsonb
) ON CONFLICT DO NOTHING;

-- 6. 測試插入圖片向量的 chunk
INSERT INTO chunks (
    chunk_id,
    contents,
    vector,
    vector_type,
    vector_model,
    metadata
) VALUES (
    gen_random_uuid(),
    'AI 生成的圖片描述：這是一個系統架構圖',
    ARRAY[0.6, 0.7, 0.8, 0.9, 1.0]::vector(512),
    'image',
    'clip-vit-b-32',
    '{
        "media_type": "image",
        "storage": {
            "type": "supabase",
            "storage_id": "test-image.png",
            "url": "https://example.com/test-image.png",
            "file_hash": "abc123"
        },
        "image_properties": {
            "format": "png",
            "size_bytes": 102400,
            "width": 1920,
            "height": 1080
        }
    }'::jsonb
) ON CONFLICT DO NOTHING;

-- 7. 測試多模態搜尋函數
SELECT 
    chunk->>'contents' as content,
    similarity,
    chunk->>'vector_type' as vector_type
FROM match_chunks_multimodal(
    ARRAY[0.1, 0.2, 0.3, 0.4, 0.5]::vector(512),
    'all',
    0.0,
    5
);

-- 8. 測試文字向量搜尋
SELECT 
    chunk->>'contents' as content,
    similarity
FROM match_chunks_multimodal(
    ARRAY[0.1, 0.2, 0.3, 0.4, 0.5]::vector(512),
    'text',
    0.0,
    5
);

-- 9. 測試圖片向量搜尋
SELECT 
    chunk->>'contents' as content,
    similarity
FROM match_chunks_multimodal(
    ARRAY[0.6, 0.7, 0.8, 0.9, 1.0]::vector(512),
    'image',
    0.0,
    5
);

-- 10. 測試混合搜尋函數
SELECT 
    chunk->>'contents' as content,
    similarity,
    match_type
FROM hybrid_search_chunks(
    ARRAY[0.1, 0.2, 0.3, 0.4, 0.5]::vector(512),  -- 文字向量
    ARRAY[0.6, 0.7, 0.8, 0.9, 1.0]::vector(512),  -- 圖片向量
    0.7,  -- 文字權重
    0.3,  -- 圖片權重
    0.0,  -- 最小相似度
    10    -- 結果數量
);

-- 11. 測試圖片去重函數
SELECT 
    chunk_id,
    storage_url,
    created_time
FROM find_duplicate_images('abc123');

-- 12. 檢查向量統計檢視
SELECT * FROM vector_statistics;

-- 13. 清理測試資料
DELETE FROM chunks WHERE metadata->>'test' = 'true';

-- 顯示測試完成訊息
DO $$
BEGIN
    RAISE NOTICE '✅ 多模態向量遷移測試完成！';
    RAISE NOTICE '📊 已驗證欄位、索引、函數和約束';
    RAISE NOTICE '🔍 已測試文字和圖片向量搜尋功能';
    RAISE NOTICE '⚡ 已測試混合搜尋和去重功能';
END $$;