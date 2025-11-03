#!/bin/bash

# 通過直接 PostgreSQL 連接設置數據庫
echo "🔧 Setting up database via direct PostgreSQL connection..."

# PostgreSQL 連接參數 (本地 Supabase)
# 嘗試不同的連接方式
DB_HOST="localhost"
DB_PORT="5432"  # 標準 PostgreSQL 端口
DB_NAME="postgres"
DB_USER="postgres"
DB_PASSWORD="your-super-secret-and-long-postgres-password"

# 如果標準端口失敗，嘗試 Supabase 常用端口
FALLBACK_PORT="54322"

# 檢查 psql 是否可用
if ! command -v psql &> /dev/null; then
    echo "❌ psql command not found"
    echo "Please install PostgreSQL client tools"
    echo "  macOS: brew install postgresql"
    echo "  Ubuntu: sudo apt-get install postgresql-client"
    exit 1
fi

# 檢查 SQL 文件是否存在
if [ ! -f "database/reset_and_recreate.sql" ]; then
    echo "❌ SQL file not found: database/reset_and_recreate.sql"
    exit 1
fi

# 驗證 SQL 腳本
echo "🔍 Validating SQL script..."
./scripts/validate-sql.sh
if [ $? -ne 0 ]; then
    echo "❌ SQL validation failed"
    exit 1
fi

# 測試 PostgreSQL 連接
echo "📡 Testing PostgreSQL connection..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT version();" > /dev/null 2>&1

if [ $? -ne 0 ]; then
    echo "⚠️  Primary connection failed, trying fallback port $FALLBACK_PORT..."
    DB_PORT=$FALLBACK_PORT
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT version();" > /dev/null 2>&1
    
    if [ $? -ne 0 ]; then
        echo "❌ Cannot connect to PostgreSQL on either port"
        echo "Tried connections:"
        echo "  Host: $DB_HOST, Port: 5432"
        echo "  Host: $DB_HOST, Port: $FALLBACK_PORT"
        echo "  Database: $DB_NAME, User: $DB_USER"
        echo ""
        echo "Please check:"
        echo "1. Supabase is running: supabase status"
        echo "2. PostgreSQL port is accessible"
        echo "3. Docker containers are running: docker ps"
        exit 1
    fi
fi

echo "✅ PostgreSQL connection successful"

# 執行 SQL 腳本
echo "🚀 Executing database setup SQL..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f database/reset_and_recreate.sql

if [ $? -eq 0 ]; then
    echo ""
    echo "🎉 Database setup completed successfully!"
    echo ""
    echo "✅ Created schemas with separated data:"
    echo "  📊 content_db: texts, chunks, chunk_tags, template_slots"
    echo "  🔍 vector_db: embeddings"
    echo "  🕸️  graph_db: graph_nodes, graph_edges"
    echo ""
    echo "✅ Created indexes for optimal performance"
    echo "✅ Created RPC functions: match_chunks, search_graph"
    echo "✅ Enabled RLS with development policies"
    echo ""
    echo "🧪 Running verification tests..."
    ./scripts/verify-setup.sh
else
    echo ""
    echo "❌ Database setup failed!"
    echo "Please check the error messages above"
    exit 1
fi