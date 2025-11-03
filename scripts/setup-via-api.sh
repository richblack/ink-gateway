#!/bin/bash

# 通過 Supabase API 設置數據庫
echo "Setting up database via Supabase API..."

SUPABASE_URL="http://localhost:8000"
ANON_KEY="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyAgCiAgICAicm9sZSI6ICJhbm9uIiwKICAgICJpc3MiOiAic3VwYWJhc2UtZGVtbyIsCiAgICAiaWF0IjogMTY0MTc2OTIwMCwKICAgICJleHAiOiAxNzk5NTM1NjAwCn0.dc_X5iR_VP_qT0zsiyj_I_OZ2T9FtRU2BBNWN8Bu4GE"
SERVICE_KEY="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyAgCiAgICAicm9sZSI6ICJzZXJ2aWNlX3JvbGUiLAogICAgImlzcyI6ICJzdXBhYmFzZS1kZW1vIiwKICAgICJpYXQiOiAxNjQxNzY5MjAwLAogICAgImV4cCI6IDE3OTk1MzU2MDAKfQ.DaYlNEoUrrEn2Ig7tqibS-PHK5vgusbcbo7X36XVt4Q"

# 檢查 Supabase 是否運行
echo "Checking Supabase connection..."
if ! curl -s -H "apikey: $ANON_KEY" "$SUPABASE_URL/rest/v1/" > /dev/null; then
    echo "Error: Cannot connect to Supabase at $SUPABASE_URL"
    echo "Please make sure Supabase is running"
    exit 1
fi

echo "✅ Supabase is running and accessible"

# 使用 service key 執行數據庫設置
echo "🔧 Setting up database schema using service key..."

# 執行 SQL 腳本
echo "📋 Executing database reset and recreation..."

# 使用 Supabase RPC 來執行 SQL
SQL_CONTENT=$(cat database/reset_and_recreate.sql)

# 通過 RPC 執行 SQL (需要 service key 權限)
curl -X POST "$SUPABASE_URL/rest/v1/rpc/exec_sql" \
  -H "apikey: $SERVICE_KEY" \
  -H "Authorization: Bearer $SERVICE_KEY" \
  -H "Content-Type: application/json" \
  -d "{\"sql\": $(echo "$SQL_CONTENT" | jq -Rs .)}" \
  > /tmp/sql_result.json 2>&1

if [ $? -eq 0 ]; then
    echo "✅ Database schema setup completed!"
else
    echo "⚠️  Direct SQL execution not available, trying alternative approach..."
    echo "📝 You may need to execute the SQL manually through Supabase Dashboard"
    echo "📁 SQL file location: database/reset_and_recreate.sql"
fi

echo ""
echo "✅ Setup completed!"
echo ""
echo "🧪 Testing with anon key:"
echo "  SUPABASE_URL=$SUPABASE_URL SUPABASE_API_KEY=$ANON_KEY INTEGRATION_TESTS=true go test -v ./clients -run TestTableVerification"
echo ""
echo "🧪 Testing with service key (if needed):"
echo "  SUPABASE_URL=$SUPABASE_URL SUPABASE_API_KEY=$SERVICE_KEY INTEGRATION_TESTS=true go test -v ./clients -run TestTableVerification"