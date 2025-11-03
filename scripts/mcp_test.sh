#!/bin/bash
# MCP Server 測試腳本

set -e

echo "🔧 開始 MCP Server 測試..."

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 建構 MCP Server
echo "🔨 建構 MCP Server..."
go build -o bin/mcp-server ./cmd/mcp-server

# 測試函數
test_mcp() {
    local name="$1"
    local request="$2"
    local timeout="${3:-10}"
    
    echo -n "測試 $name... "
    
    response=$(echo "$request" | timeout "$timeout" ./bin/mcp-server 2>/dev/null || echo "TIMEOUT")
    
    if [ "$response" = "TIMEOUT" ]; then
        echo -e "${RED}❌ 超時${NC}"
        return 1
    elif echo "$response" | jq -e '.result' >/dev/null 2>&1; then
        echo -e "${GREEN}✅ 通過${NC}"
        return 0
    elif echo "$response" | jq -e '.error' >/dev/null 2>&1; then
        echo -e "${YELLOW}⚠️  錯誤回應${NC}"
        echo "錯誤: $(echo "$response" | jq -r '.error.message')"
        return 1
    else
        echo -e "${RED}❌ 無效回應${NC}"
        echo "回應: $response"
        return 1
    fi
}

# 1. 測試初始化
test_mcp "初始化" '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {}
}'

# 2. 測試工具列表
test_mcp "工具列表" '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/list",
    "params": {}
}'

# 3. 測試搜尋工具
test_mcp "搜尋工具" '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
        "name": "ink_search_chunks",
        "arguments": {
            "query": "測試",
            "search_type": "hybrid",
            "limit": 5
        }
    }
}' 15

# 4. 測試圖片分析工具
test_mcp "圖片分析工具" '{
    "jsonrpc": "2.0",
    "id": 4,
    "method": "tools/call",
    "params": {
        "name": "ink_analyze_image",
        "arguments": {
            "image_url": "https://via.placeholder.com/150",
            "detail_level": "medium"
        }
    }
}' 20

# 5. 測試資源列表
test_mcp "資源列表" '{
    "jsonrpc": "2.0",
    "id": 5,
    "method": "resources/list",
    "params": {}
}'

# 6. 測試提示列表
test_mcp "提示列表" '{
    "jsonrpc": "2.0",
    "id": 6,
    "method": "prompts/list",
    "params": {}
}'

# 7. 測試搜尋提示
test_mcp "搜尋提示" '{
    "jsonrpc": "2.0",
    "id": 7,
    "method": "prompts/get",
    "params": {
        "name": "ink_search_assistant",
        "arguments": {
            "search_context": "我想找關於機器學習的資料",
            "content_type": "both"
        }
    }
}'

# 8. 測試批次處理工具
test_mcp "批次處理工具" '{
    "jsonrpc": "2.0",
    "id": 8,
    "method": "tools/call",
    "params": {
        "name": "ink_batch_process_images",
        "arguments": {
            "folder_path": "/tmp/test-images",
            "auto_analyze": true,
            "concurrency": 2
        }
    }
}' 15

# 9. 測試投影片推薦工具
test_mcp "投影片推薦工具" '{
    "jsonrpc": "2.0",
    "id": 9,
    "method": "tools/call",
    "params": {
        "name": "ink_get_images_for_slides",
        "arguments": {
            "slide_title": "人工智慧概述",
            "slide_content": "介紹機器學習和深度學習的基本概念",
            "max_suggestions": 3
        }
    }
}' 15

# 10. 測試重複檢測工具
test_mcp "重複檢測工具" '{
    "jsonrpc": "2.0",
    "id": 10,
    "method": "tools/call",
    "params": {
        "name": "find_duplicates",
        "arguments": {
            "similarity_threshold": 0.95,
            "min_group_size": 2
        }
    }
}' 15

echo ""
echo -e "${GREEN}🎉 MCP Server 測試完成！${NC}"