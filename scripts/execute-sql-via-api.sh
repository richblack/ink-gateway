#!/bin/bash

# 透過 Supabase API 執行 SQL 腳本
echo "🔧 Executing SQL via Supabase API..."

SUPABASE_URL="http://localhost:8000"
SERVICE_KEY="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyAgCiAgICAicm9sZSI6ICJzZXJ2aWNlX3JvbGUiLAogICAgImlzcyI6ICJzdXBhYmFzZS1kZW1vIiwKICAgICJpYXQiOiAxNjQxNzY5MjAwLAogICAgImV4cCI6IDE3OTk1MzU2MDAKfQ.DaYlNEoUrrEn2Ig7tqibS-PHK5vgusbcbo7X36XVt4Q"

# 檢查 SQL 文件是否存在
if [ ! -f "database/reset_and_recreate.sql" ]; then
    echo "❌ SQL file not found: database/reset_and_recreate.sql"
    exit 1
fi

# 檢查 Supabase 是否運行
echo "📡 Checking Supabase connection..."
if ! curl -s -H "apikey: $SERVICE_KEY" "$SUPABASE_URL/rest/v1/" > /dev/null; then
    echo "❌ Cannot connect to Supabase at $SUPABASE_URL"
    echo "Please make sure Supabase is running"
    exit 1
fi

echo "✅ Supabase is accessible"

# 讀取 SQL 文件內容
echo "📖 Reading SQL file..."
SQL_CONTENT=$(cat database/reset_and_recreate.sql)

# 透過 Supabase RPC 執行 SQL
echo "🚀 Executing SQL via Supabase API..."

# 使用 Supabase 的 rpc endpoint 來執行原始 SQL
# 注意：這需要創建一個 RPC 函數來執行任意 SQL
curl -X POST "$SUPABASE_URL/rest/v1/rpc/execute_sql" \
  -H "apikey: $SERVICE_KEY" \
  -H "Authorization: Bearer $SERVICE_KEY" \
  -H "Content-Type: application/json" \
  -d "{\"sql_query\": $(echo "$SQL_CONTENT" | jq -Rs .)}" \
  --fail --silent --show-error

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ SQL executed successfully!"
    echo ""
    echo "🎉 Database setup completed with separated schemas:"
    echo "  📊 content_db: texts, chunks, chunk_tags, template_slots"
    echo "  🔍 vector_db: embeddings"
    echo "  🕸️  graph_db: graph_nodes, graph_edges"
    echo ""
    echo "🧪 You can now run verification tests:"
    echo "  ./scripts/verify-setup.sh"
else
    echo ""
    echo "❌ SQL execution failed!"
    echo "This might be because the execute_sql RPC function doesn't exist."
    echo "Let's try an alternative approach..."
    
    # 備用方案：分段執行 SQL
    echo ""
    echo "🔄 Trying alternative approach: executing SQL in segments..."
    
    # 先創建 schemas
    echo "📁 Creating schemas..."
    curl -X POST "$SUPABASE_URL/rest/v1/rpc/exec" \
      -H "apikey: $SERVICE_KEY" \
      -H "Authorization: Bearer $SERVICE_KEY" \
      -H "Content-Type: application/json" \
      -d '{"sql": "CREATE SCHEMA IF NOT EXISTS content_db; CREATE SCHEMA IF NOT EXISTS vector_db; CREATE SCHEMA IF NOT EXISTS graph_db;"}' \
      --silent
    
    if [ $? -eq 0 ]; then
        echo "✅ Schemas created successfully!"
        echo ""
        echo "⚠️  Please run the full SQL script manually in Supabase Dashboard"
        echo "   or use a PostgreSQL client to execute: database/reset_and_recreate.sql"
    else
        echo "❌ Alternative approach also failed"
        echo ""
        echo "📋 Manual setup required:"
        echo "1. Open Supabase Dashboard"
        echo "2. Go to SQL Editor"
        echo "3. Copy and paste the content of database/reset_and_recreate.sql"
        echo "4. Execute the SQL"
    fi
fi