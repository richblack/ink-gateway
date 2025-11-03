#!/bin/bash

# 恢復 Supabase 客戶端的 endpoint 來使用標準的表格名稱
echo "🔄 Reverting Supabase client endpoints to use standard table names..."

# 備份原始文件
cp clients/supabase.go clients/supabase.go.backup

# 恢復標準的 endpoint 名稱
sed -i '' 's|"/content_db\.texts"|"/texts"|g' clients/supabase.go
sed -i '' 's|"/content_db\.chunks"|"/chunks"|g' clients/supabase.go
sed -i '' 's|"/content_db\.chunk_tags"|"/chunk_tags"|g' clients/supabase.go
sed -i '' 's|"/vector_db\.embeddings"|"/embeddings"|g' clients/supabase.go
sed -i '' 's|"/graph_db\.graph_nodes"|"/graph_nodes"|g' clients/supabase.go
sed -i '' 's|"/graph_db\.graph_edges"|"/graph_edges"|g' clients/supabase.go

# 恢復 RPC 函數調用
sed -i '' 's|"/rpc/vector_db\.match_chunks"|"/rpc/match_chunks"|g' clients/supabase.go
sed -i '' 's|"/rpc/graph_db\.search_graph"|"/rpc/search_graph"|g' clients/supabase.go

echo "✅ Client endpoints reverted successfully!"
echo ""
echo "📋 Reverted schemas:"
echo "  📊 /content_db.texts → /texts"
echo "  📊 /content_db.chunks → /chunks" 
echo "  📊 /content_db.chunk_tags → /chunk_tags"
echo "  🔍 /vector_db.embeddings → /embeddings"
echo "  🕸️  /graph_db.graph_nodes → /graph_nodes"
echo "  🕸️  /graph_db.graph_edges → /graph_edges"
echo ""
echo "🧪 Testing compilation..."
go build ./clients

if [ $? -eq 0 ]; then
    echo "✅ Client compiles successfully!"
    rm clients/supabase.go.backup
else
    echo "❌ Compilation failed, restoring backup..."
    mv clients/supabase.go.backup clients/supabase.go
    exit 1
fi