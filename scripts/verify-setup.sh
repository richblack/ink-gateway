#!/bin/bash

# 驗證 Supabase 設置的腳本
echo "🔍 Verifying Supabase setup..."

SUPABASE_URL="http://localhost:8000"
# 使用 service key 進行測試，有更高權限
SERVICE_KEY="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyAgCiAgICAicm9sZSI6ICJzZXJ2aWNlX3JvbGUiLAogICAgImlzcyI6ICJzdXBhYmFzZS1kZW1vIiwKICAgICJpYXQiOiAxNjQxNzY5MjAwLAogICAgImV4cCI6IDE3OTk1MzU2MDAKfQ.DaYlNEoUrrEn2Ig7tqibS-PHK5vgusbcbo7X36XVt4Q"

echo "📋 Running table verification tests..."
SUPABASE_URL=$SUPABASE_URL SUPABASE_API_KEY=$SERVICE_KEY INTEGRATION_TESTS=true go test -v ./clients -run TestTableVerification

if [ $? -eq 0 ]; then
    echo ""
    echo "🎉 All table verification tests passed!"
    echo ""
    echo "📋 Running complete graph workflow test..."
    SUPABASE_URL=$SUPABASE_URL SUPABASE_API_KEY=$SERVICE_KEY INTEGRATION_TESTS=true go test -v ./clients -run TestFullGraphWorkflow
    
    if [ $? -eq 0 ]; then
        echo ""
        echo "🎉 Complete workflow test passed!"
        echo ""
        echo "✅ Supabase setup is complete and working!"
        echo ""
        echo "🚀 You can now run all integration tests:"
        echo "  SUPABASE_URL=$SUPABASE_URL SUPABASE_API_KEY=$SERVICE_KEY INTEGRATION_TESTS=true go test -v ./clients"
    else
        echo ""
        echo "❌ Complete workflow test failed"
        echo "Please check the error messages above"
    fi
else
    echo ""
    echo "❌ Table verification failed"
    echo "Please make sure all required tables are created:"
    echo "  - texts"
    echo "  - chunks" 
    echo "  - graph_nodes"
    echo "  - graph_edges"
    echo "  - embeddings"
    echo "  - chunk_tags (optional)"
    echo "  - template_slots (optional)"
fi