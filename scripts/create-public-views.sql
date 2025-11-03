-- 在 public schema 中創建視圖來暴露其他 schema 的表格
-- 這樣 Supabase API 就可以訪問它們了

-- 創建 texts 視圖
CREATE OR REPLACE VIEW public.texts AS 
SELECT * FROM content_db.texts;

-- 創建 chunks 視圖
CREATE OR REPLACE VIEW public.chunks AS 
SELECT * FROM content_db.chunks;

-- 創建 chunk_tags 視圖
CREATE OR REPLACE VIEW public.chunk_tags AS 
SELECT * FROM content_db.chunk_tags;

-- 創建 template_slots 視圖
CREATE OR REPLACE VIEW public.template_slots AS 
SELECT * FROM content_db.template_slots;

-- 創建 embeddings 視圖
CREATE OR REPLACE VIEW public.embeddings AS 
SELECT * FROM vector_db.embeddings;

-- 創建 graph_nodes 視圖
CREATE OR REPLACE VIEW public.graph_nodes AS 
SELECT * FROM graph_db.graph_nodes;

-- 創建 graph_edges 視圖
CREATE OR REPLACE VIEW public.graph_edges AS 
SELECT * FROM graph_db.graph_edges;

-- 創建可更新的規則，讓視圖支持 INSERT/UPDATE/DELETE

-- texts 視圖規則
CREATE OR REPLACE RULE texts_insert AS ON INSERT TO public.texts 
DO INSTEAD INSERT INTO content_db.texts VALUES (NEW.*);

CREATE OR REPLACE RULE texts_update AS ON UPDATE TO public.texts 
DO INSTEAD UPDATE content_db.texts SET 
    content = NEW.content,
    title = NEW.title,
    status = NEW.status,
    updated_at = NEW.updated_at
WHERE id = OLD.id;

CREATE OR REPLACE RULE texts_delete AS ON DELETE TO public.texts 
DO INSTEAD DELETE FROM content_db.texts WHERE id = OLD.id;

-- chunks 視圖規則
CREATE OR REPLACE RULE chunks_insert AS ON INSERT TO public.chunks 
DO INSTEAD INSERT INTO content_db.chunks VALUES (NEW.*);

CREATE OR REPLACE RULE chunks_update AS ON UPDATE TO public.chunks 
DO INSTEAD UPDATE content_db.chunks SET 
    text_id = NEW.text_id,
    content = NEW.content,
    is_template = NEW.is_template,
    is_slot = NEW.is_slot,
    parent_chunk_id = NEW.parent_chunk_id,
    template_chunk_id = NEW.template_chunk_id,
    slot_value = NEW.slot_value,
    indent_level = NEW.indent_level,
    sequence_number = NEW.sequence_number,
    metadata = NEW.metadata,
    updated_at = NEW.updated_at
WHERE id = OLD.id;

CREATE OR REPLACE RULE chunks_delete AS ON DELETE TO public.chunks 
DO INSTEAD DELETE FROM content_db.chunks WHERE id = OLD.id;

-- chunk_tags 視圖規則
CREATE OR REPLACE RULE chunk_tags_insert AS ON INSERT TO public.chunk_tags 
DO INSTEAD INSERT INTO content_db.chunk_tags VALUES (NEW.*);

CREATE OR REPLACE RULE chunk_tags_delete AS ON DELETE TO public.chunk_tags 
DO INSTEAD DELETE FROM content_db.chunk_tags WHERE id = OLD.id;

-- embeddings 視圖規則
CREATE OR REPLACE RULE embeddings_insert AS ON INSERT TO public.embeddings 
DO INSTEAD INSERT INTO vector_db.embeddings VALUES (NEW.*);

CREATE OR REPLACE RULE embeddings_delete AS ON DELETE TO public.embeddings 
DO INSTEAD DELETE FROM vector_db.embeddings WHERE id = OLD.id;

-- graph_nodes 視圖規則
CREATE OR REPLACE RULE graph_nodes_insert AS ON INSERT TO public.graph_nodes 
DO INSTEAD INSERT INTO graph_db.graph_nodes VALUES (NEW.*);

CREATE OR REPLACE RULE graph_nodes_update AS ON UPDATE TO public.graph_nodes 
DO INSTEAD UPDATE graph_db.graph_nodes SET 
    chunk_id = NEW.chunk_id,
    entity_name = NEW.entity_name,
    entity_type = NEW.entity_type,
    properties = NEW.properties
WHERE id = OLD.id;

CREATE OR REPLACE RULE graph_nodes_delete AS ON DELETE TO public.graph_nodes 
DO INSTEAD DELETE FROM graph_db.graph_nodes WHERE id = OLD.id;

-- graph_edges 視圖規則
CREATE OR REPLACE RULE graph_edges_insert AS ON INSERT TO public.graph_edges 
DO INSTEAD INSERT INTO graph_db.graph_edges VALUES (NEW.*);

CREATE OR REPLACE RULE graph_edges_update AS ON UPDATE TO public.graph_edges 
DO INSTEAD UPDATE graph_db.graph_edges SET 
    source_node_id = NEW.source_node_id,
    target_node_id = NEW.target_node_id,
    relationship_type = NEW.relationship_type,
    properties = NEW.properties
WHERE id = OLD.id;

CREATE OR REPLACE RULE graph_edges_delete AS ON DELETE TO public.graph_edges 
DO INSTEAD DELETE FROM graph_db.graph_edges WHERE id = OLD.id;

-- 授權視圖給 Supabase 角色
GRANT ALL ON public.texts TO anon, authenticated, service_role;
GRANT ALL ON public.chunks TO anon, authenticated, service_role;
GRANT ALL ON public.chunk_tags TO anon, authenticated, service_role;
GRANT ALL ON public.template_slots TO anon, authenticated, service_role;
GRANT ALL ON public.embeddings TO anon, authenticated, service_role;
GRANT ALL ON public.graph_nodes TO anon, authenticated, service_role;
GRANT ALL ON public.graph_edges TO anon, authenticated, service_role;

-- 顯示成功訊息
DO $$
BEGIN
    RAISE NOTICE '✅ Public views created successfully!';
    RAISE NOTICE '📊 Views: texts, chunks, chunk_tags, template_slots';
    RAISE NOTICE '🔍 Views: embeddings';
    RAISE NOTICE '🕸️  Views: graph_nodes, graph_edges';
    RAISE NOTICE '🔄 All views support INSERT/UPDATE/DELETE operations';
    RAISE NOTICE '🚀 Supabase API can now access all schemas through public views!';
END $$;