-- 重置並重新創建數據庫表格 (使用分離的 Schema)
-- 這個腳本會刪除所有現有的表格和 schema，然後重新創建

-- ==========================================
-- 第一步：清理現有的結構
-- ==========================================

-- 刪除現有的 RLS 政策
DROP POLICY IF EXISTS "Allow all operations" ON public.texts;
DROP POLICY IF EXISTS "Allow all operations" ON public.chunks;
DROP POLICY IF EXISTS "Allow all operations" ON public.chunk_tags;
DROP POLICY IF EXISTS "Allow all operations" ON public.template_slots;
DROP POLICY IF EXISTS "Allow all operations" ON public.embeddings;
DROP POLICY IF EXISTS "Allow all operations" ON public.graph_nodes;
DROP POLICY IF EXISTS "Allow all operations" ON public.graph_edges;

DROP POLICY IF EXISTS "Allow all operations" ON content_db.texts;
DROP POLICY IF EXISTS "Allow all operations" ON content_db.chunks;
DROP POLICY IF EXISTS "Allow all operations" ON content_db.chunk_tags;
DROP POLICY IF EXISTS "Allow all operations" ON content_db.template_slots;
DROP POLICY IF EXISTS "Allow all operations" ON vector_db.embeddings;
DROP POLICY IF EXISTS "Allow all operations" ON graph_db.graph_nodes;
DROP POLICY IF EXISTS "Allow all operations" ON graph_db.graph_edges;

-- 刪除現有的函數
DROP FUNCTION IF EXISTS public.match_chunks(vector, float, int);
DROP FUNCTION IF EXISTS public.search_graph(text, int, int);
DROP FUNCTION IF EXISTS vector_db.match_chunks(vector, float, int);
DROP FUNCTION IF EXISTS graph_db.search_graph(text, int, int);

-- 刪除現有的表格 (按依賴順序)
DROP TABLE IF EXISTS public.graph_edges CASCADE;
DROP TABLE IF EXISTS public.graph_nodes CASCADE;
DROP TABLE IF EXISTS public.embeddings CASCADE;
DROP TABLE IF EXISTS public.template_slots CASCADE;
DROP TABLE IF EXISTS public.chunk_tags CASCADE;
DROP TABLE IF EXISTS public.chunks CASCADE;
DROP TABLE IF EXISTS public.texts CASCADE;

DROP TABLE IF EXISTS graph_db.graph_edges CASCADE;
DROP TABLE IF EXISTS graph_db.graph_nodes CASCADE;
DROP TABLE IF EXISTS vector_db.embeddings CASCADE;
DROP TABLE IF EXISTS content_db.template_slots CASCADE;
DROP TABLE IF EXISTS content_db.chunk_tags CASCADE;
DROP TABLE IF EXISTS content_db.chunks CASCADE;
DROP TABLE IF EXISTS content_db.texts CASCADE;

-- 刪除自定義 schema
DROP SCHEMA IF EXISTS content_db CASCADE;
DROP SCHEMA IF EXISTS vector_db CASCADE;
DROP SCHEMA IF EXISTS graph_db CASCADE;

-- ==========================================
-- 第二步：重新創建正確的結構 (分離的 Schema)
-- ==========================================

-- 創建不同的 schema 來分離不同類型的資料
CREATE SCHEMA IF NOT EXISTS content_db;     -- 關聯資料庫 (文字和 chunks)
CREATE SCHEMA IF NOT EXISTS vector_db;      -- 向量資料庫 (embeddings)
CREATE SCHEMA IF NOT EXISTS graph_db;       -- 圖形資料庫 (nodes and edges)

-- 啟用必要的擴展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS vector;

-- ==========================================
-- Content DB Schema (關聯資料庫)
-- ==========================================

-- 文字表
CREATE TABLE content_db.texts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content TEXT NOT NULL,
    title VARCHAR(255),
    status VARCHAR(50) DEFAULT 'processing',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Chunks 表
CREATE TABLE content_db.chunks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    text_id UUID NOT NULL REFERENCES content_db.texts(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_template BOOLEAN DEFAULT FALSE,
    is_slot BOOLEAN DEFAULT FALSE,
    parent_chunk_id UUID REFERENCES content_db.chunks(id) ON DELETE CASCADE,
    template_chunk_id UUID REFERENCES content_db.chunks(id) ON DELETE SET NULL,
    slot_value TEXT,
    indent_level INTEGER DEFAULT 0,
    sequence_number INTEGER,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Chunk 標籤關係表
CREATE TABLE content_db.chunk_tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chunk_id UUID NOT NULL REFERENCES content_db.chunks(id) ON DELETE CASCADE,
    tag_chunk_id UUID NOT NULL REFERENCES content_db.chunks(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(chunk_id, tag_chunk_id)
);

-- Template slots 表
CREATE TABLE content_db.template_slots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_chunk_id UUID NOT NULL REFERENCES content_db.chunks(id) ON DELETE CASCADE,
    slot_chunk_id UUID NOT NULL REFERENCES content_db.chunks(id) ON DELETE CASCADE,
    slot_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ==========================================
-- Vector DB Schema (向量資料庫)
-- ==========================================

-- Embeddings 表
CREATE TABLE vector_db.embeddings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chunk_id UUID NOT NULL,
    vector vector(1536), -- OpenAI embeddings 維度
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    FOREIGN KEY (chunk_id) REFERENCES content_db.chunks(id) ON DELETE CASCADE
);

-- ==========================================
-- Graph DB Schema (圖形資料庫)
-- ==========================================

-- Graph nodes 表
CREATE TABLE graph_db.graph_nodes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chunk_id UUID NOT NULL,
    entity_name VARCHAR(255) NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    properties JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    FOREIGN KEY (chunk_id) REFERENCES content_db.chunks(id) ON DELETE CASCADE
);

-- Graph edges 表
CREATE TABLE graph_db.graph_edges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_node_id UUID NOT NULL REFERENCES graph_db.graph_nodes(id) ON DELETE CASCADE,
    target_node_id UUID NOT NULL REFERENCES graph_db.graph_nodes(id) ON DELETE CASCADE,
    relationship_type VARCHAR(100) NOT NULL,
    properties JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ==========================================
-- 索引優化
-- ==========================================

-- Content DB 索引
CREATE INDEX idx_chunks_text_id ON content_db.chunks(text_id);
CREATE INDEX idx_chunks_parent_id ON content_db.chunks(parent_chunk_id);
CREATE INDEX idx_chunks_template_id ON content_db.chunks(template_chunk_id);
CREATE INDEX idx_chunks_content_search ON content_db.chunks USING gin(to_tsvector('english', content));
CREATE INDEX idx_chunk_tags_chunk_id ON content_db.chunk_tags(chunk_id);
CREATE INDEX idx_chunk_tags_tag_chunk_id ON content_db.chunk_tags(tag_chunk_id);

-- Vector DB 索引
CREATE INDEX idx_embeddings_chunk_id ON vector_db.embeddings(chunk_id);
CREATE INDEX embeddings_vector_idx ON vector_db.embeddings 
USING ivfflat (vector vector_cosine_ops) WITH (lists = 100);

-- Graph DB 索引
CREATE INDEX idx_graph_nodes_chunk_id ON graph_db.graph_nodes(chunk_id);
CREATE INDEX idx_graph_nodes_entity_name ON graph_db.graph_nodes(entity_name);
CREATE INDEX idx_graph_nodes_entity_type ON graph_db.graph_nodes(entity_type);
CREATE INDEX idx_graph_edges_source ON graph_db.graph_edges(source_node_id);
CREATE INDEX idx_graph_edges_target ON graph_db.graph_edges(target_node_id);
CREATE INDEX idx_graph_edges_relationship ON graph_db.graph_edges(relationship_type);

-- ==========================================
-- RPC 函數
-- ==========================================

-- 向量相似性搜尋函數
CREATE OR REPLACE FUNCTION public.match_chunks(
    query_embedding vector(1536),
    match_threshold float DEFAULT 0.0,
    match_count int DEFAULT 50
)
RETURNS TABLE (
    chunk jsonb,
    similarity float
)
LANGUAGE sql STABLE
AS $$
    SELECT 
        to_jsonb(c.*) as chunk,
        1 - (e.vector <=> query_embedding) as similarity
    FROM vector_db.embeddings e
    JOIN content_db.chunks c ON e.chunk_id = c.id
    WHERE 1 - (e.vector <=> query_embedding) > match_threshold
    ORDER BY e.vector <=> query_embedding
    LIMIT match_count;
$$;

-- 圖形搜尋函數
CREATE OR REPLACE FUNCTION public.search_graph(
    entity_name text,
    max_depth int DEFAULT 3,
    result_limit int DEFAULT 50
)
RETURNS TABLE (
    nodes jsonb,
    edges jsonb
)
LANGUAGE sql STABLE
AS $$
    WITH RECURSIVE graph_traversal AS (
        -- 起始節點
        SELECT 
            n.id,
            n.entity_name,
            n.entity_type,
            n.properties,
            n.chunk_id,
            n.created_at,
            0 as depth
        FROM graph_db.graph_nodes n
        WHERE n.entity_name = search_graph.entity_name
        
        UNION ALL
        
        -- 遞歸遍歷
        SELECT 
            n.id,
            n.entity_name,
            n.entity_type,
            n.properties,
            n.chunk_id,
            n.created_at,
            gt.depth + 1
        FROM graph_db.graph_nodes n
        JOIN graph_db.graph_edges e ON (n.id = e.source_node_id OR n.id = e.target_node_id)
        JOIN graph_traversal gt ON (
            (e.source_node_id = gt.id AND n.id = e.target_node_id) OR
            (e.target_node_id = gt.id AND n.id = e.source_node_id)
        )
        WHERE gt.depth < search_graph.max_depth
    ),
    found_nodes AS (
        SELECT DISTINCT * FROM graph_traversal
        LIMIT result_limit
    ),
    found_edges AS (
        SELECT DISTINCT e.*
        FROM graph_db.graph_edges e
        WHERE e.source_node_id IN (SELECT id FROM found_nodes)
           OR e.target_node_id IN (SELECT id FROM found_nodes)
    )
    SELECT 
        COALESCE(jsonb_agg(to_jsonb(fn.*)) FILTER (WHERE fn.id IS NOT NULL), '[]'::jsonb) as nodes,
        COALESCE(jsonb_agg(to_jsonb(fe.*)) FILTER (WHERE fe.id IS NOT NULL), '[]'::jsonb) as edges
    FROM found_nodes fn
    FULL OUTER JOIN found_edges fe ON true;
$$;

-- ==========================================
-- Row Level Security (RLS) 設置
-- ==========================================

-- 啟用 RLS
ALTER TABLE content_db.texts ENABLE ROW LEVEL SECURITY;
ALTER TABLE content_db.chunks ENABLE ROW LEVEL SECURITY;
ALTER TABLE content_db.chunk_tags ENABLE ROW LEVEL SECURITY;
ALTER TABLE content_db.template_slots ENABLE ROW LEVEL SECURITY;
ALTER TABLE vector_db.embeddings ENABLE ROW LEVEL SECURITY;
ALTER TABLE graph_db.graph_nodes ENABLE ROW LEVEL SECURITY;
ALTER TABLE graph_db.graph_edges ENABLE ROW LEVEL SECURITY;

-- 創建允許所有操作的政策 (開發環境)
CREATE POLICY "Allow all operations" ON content_db.texts FOR ALL USING (true);
CREATE POLICY "Allow all operations" ON content_db.chunks FOR ALL USING (true);
CREATE POLICY "Allow all operations" ON content_db.chunk_tags FOR ALL USING (true);
CREATE POLICY "Allow all operations" ON content_db.template_slots FOR ALL USING (true);
CREATE POLICY "Allow all operations" ON vector_db.embeddings FOR ALL USING (true);
CREATE POLICY "Allow all operations" ON graph_db.graph_nodes FOR ALL USING (true);
CREATE POLICY "Allow all operations" ON graph_db.graph_edges FOR ALL USING (true);

-- ==========================================
-- 授權
-- ==========================================

-- 授權給 anon 和 authenticated 角色
GRANT USAGE ON SCHEMA content_db TO anon, authenticated;
GRANT USAGE ON SCHEMA vector_db TO anon, authenticated;
GRANT USAGE ON SCHEMA graph_db TO anon, authenticated;

GRANT ALL ON ALL TABLES IN SCHEMA content_db TO anon, authenticated;
GRANT ALL ON ALL TABLES IN SCHEMA vector_db TO anon, authenticated;
GRANT ALL ON ALL TABLES IN SCHEMA graph_db TO anon, authenticated;

GRANT ALL ON ALL SEQUENCES IN SCHEMA content_db TO anon, authenticated;
GRANT ALL ON ALL SEQUENCES IN SCHEMA vector_db TO anon, authenticated;
GRANT ALL ON ALL SEQUENCES IN SCHEMA graph_db TO anon, authenticated;

GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO anon, authenticated;

-- ==========================================
-- 完成訊息
-- ==========================================

-- 插入一條測試記錄來驗證設置
DO $$
BEGIN
    RAISE NOTICE '✅ Database reset and recreation completed successfully!';
    RAISE NOTICE '📋 Created tables: texts, chunks, chunk_tags, template_slots, embeddings, graph_nodes, graph_edges';
    RAISE NOTICE '🔍 Created indexes for optimal performance';
    RAISE NOTICE '⚡ Created RPC functions: match_chunks, search_graph';
    RAISE NOTICE '🔒 Enabled RLS with permissive policies for development';
    RAISE NOTICE '🚀 Ready for integration testing!';
END $$;