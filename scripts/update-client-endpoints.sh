#!/bin/bash

# 更新 Supabase 客戶端的 endpoint 來使用正確的 schema
echo "🔧 Updating Supabase client endpoints to use correct schemas..."

# 備份原始文件
cp clients/supabase.go clients/supabase.go.backup

# 更新 texts endpoints (content_db schema)
sed -i '' 's|"/texts"|"/content_db.texts"|g' clients/supabase.go

# 更新 chunks endpoints (content_db schema)  
sed -i '' 's|"/chunks"|"/content_db.chunks"|g' clients/supabase.go

# 更新 chunk_tags endpoints (content_db schema)
sed -i '' 's|"/chunk_tags"|"/content_db.chunk_tags"|g' clients/supabase.go

# 更新 embeddings endpoints (vector_db schema)
sed -i '' 's|"/embeddings"|"/vector_db.embeddings"|g' clients/supabase.go

# 更新 graph_nodes endpoints (graph_db schema)
sed -i '' 's|"/graph_nodes"|"/graph_db.graph_nodes"|g' clients/supabase.go

# 更新 graph_edges endpoints (graph_db schema)
sed -i '' 's|"/graph_edges"|"/graph_db.graph_edges"|g' clients/supabase.go

# 更新 RPC 函數調用
sed -i '' 's|"/rpc/match_chunks"|"/rpc/vector_db.match_chunks"|g' clients/supabase.go
sed -i '' 's|"/rpc/search_graph"|"/rpc/graph_db.search_graph"|g' clients/supabase.go

echo "✅ Client endpoints updated successfully!"
echo ""
echo "📋 Updated schemas:"
echo "  📊 /texts → /content_db.texts"
echo "  📊 /chunks → /content_db.chunks" 
echo "  📊 /chunk_tags → /content_db.chunk_tags"
echo "  🔍 /embeddings → /vector_db.embeddings"
echo "  🕸️  /graph_nodes → /graph_db.graph_nodes"
echo "  🕸️  /graph_edges → /graph_db.graph_edges"
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