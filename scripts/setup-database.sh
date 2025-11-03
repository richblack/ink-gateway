#!/bin/bash

# 設置數據庫腳本
echo "Setting up database schemas and tables..."

# 檢查 Supabase 是否運行
if ! curl -s http://localhost:8000/health > /dev/null; then
    echo "Error: Supabase is not running on localhost:8000"
    echo "Please start Supabase with: supabase start"
    exit 1
fi

# 執行數據庫初始化 SQL
echo "Executing database initialization..."

# 使用 psql 連接到本地 Supabase PostgreSQL
# 本地 Supabase 連接參數
DB_HOST="db"
DB_PORT="5432"
DB_NAME="postgres"
DB_USER="postgres"
DB_PASSWORD="postgres"

# 執行 SQL 文件
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f database/init.sql

if [ $? -eq 0 ]; then
    echo "Database setup completed successfully!"
    echo ""
    echo "Created schemas:"
    echo "  - content_db (texts, chunks, tags)"
    echo "  - vector_db (embeddings)"
    echo "  - graph_db (nodes, edges)"
    echo ""
    echo "You can now run integration tests with:"
    echo "  INTEGRATION_TESTS=true go test -v ./clients -run TestGraphIntegration"
else
    echo "Database setup failed!"
    exit 1
fi