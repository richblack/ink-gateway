-- 創建 Supabase 需要的角色
-- 這個腳本創建 anon 和 authenticated 角色，如果它們不存在的話

-- 創建 anon 角色
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'anon') THEN
        CREATE ROLE anon NOLOGIN;
        RAISE NOTICE 'Created role: anon';
    ELSE
        RAISE NOTICE 'Role anon already exists';
    END IF;
END
$$;

-- 創建 authenticated 角色
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'authenticated') THEN
        CREATE ROLE authenticated NOLOGIN;
        RAISE NOTICE 'Created role: authenticated';
    ELSE
        RAISE NOTICE 'Role authenticated already exists';
    END IF;
END
$$;

-- 創建 service_role 角色
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'service_role') THEN
        CREATE ROLE service_role NOLOGIN;
        RAISE NOTICE 'Created role: service_role';
    ELSE
        RAISE NOTICE 'Role service_role already exists';
    END IF;
END
$$;

-- 授權給新創建的角色
GRANT USAGE ON SCHEMA content_db TO anon, authenticated, service_role;
GRANT USAGE ON SCHEMA vector_db TO anon, authenticated, service_role;
GRANT USAGE ON SCHEMA graph_db TO anon, authenticated, service_role;

GRANT ALL ON ALL TABLES IN SCHEMA content_db TO anon, authenticated, service_role;
GRANT ALL ON ALL TABLES IN SCHEMA vector_db TO anon, authenticated, service_role;
GRANT ALL ON ALL TABLES IN SCHEMA graph_db TO anon, authenticated, service_role;

GRANT ALL ON ALL SEQUENCES IN SCHEMA content_db TO anon, authenticated, service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA vector_db TO anon, authenticated, service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA graph_db TO anon, authenticated, service_role;

GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO anon, authenticated, service_role;

-- 顯示成功訊息
DO $$
BEGIN
    RAISE NOTICE '✅ Supabase roles created and permissions granted!';
    RAISE NOTICE '🔑 Roles: anon, authenticated, service_role';
    RAISE NOTICE '📊 Granted access to: content_db, vector_db, graph_db';
END $$;