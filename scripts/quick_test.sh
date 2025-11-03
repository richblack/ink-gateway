#!/bin/bash
# 快速測試腳本 - 用於開發階段的快速驗證

set -e

echo "⚡ 快速測試開始"

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

BASE_URL="http://localhost:8080"

# 檢查服務是否運行
check_service() {
    echo -n "檢查 Ink-Gateway 服務... "
    if curl -s "$BASE_URL/health" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ 運行中${NC}"
        return 0
    else
        echo -e "${RED}❌ 未運行${NC}"
        echo "請先啟動 Ink-Gateway: make run"
        exit 1
    fi
}

# 快速 API 測試
quick_api_test() {
    echo -n "測試基本 API... "
    if curl -s "$BASE_URL/api/v1/chunks" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ 通過${NC}"
    else
        echo -e "${RED}❌ 失敗${NC}"
        return 1
    fi
}

# 快速圖片上傳測試
quick_upload_test() {
    echo -n "測試圖片上傳... "
    local png_data="iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="
    
    local response=$(curl -s -X POST "$BASE_URL/api/v1/media/upload" \
        -H "Content-Type: application/json" \
        -d "{\"image_data\":\"$png_data\",\"filename\":\"quick_test.png\"}")
    
    if echo "$response" | jq -e '.chunk_id' >/dev/null 2>&1; then
        echo -e "${GREEN}✅ 通過${NC}"
        return 0
    else
        echo -e "${RED}❌ 失敗${NC}"
        echo "回應: $response"
        return 1
    fi
}

# 快速搜尋測試
quick_search_test() {
    echo -n "測試多模態搜尋... "
    local response=$(curl -s -X POST "$BASE_URL/api/v1/search/multimodal" \
        -H "Content-Type: application/json" \
        -d '{"text_query":"test","search_type":"hybrid","limit":3}')
    
    if echo "$response" | jq -e '.results' >/dev/null 2>&1; then
        echo -e "${GREEN}✅ 通過${NC}"
        return 0
    else
        echo -e "${RED}❌ 失敗${NC}"
        echo "回應: $response"
        return 1
    fi
}

# 快速 MCP 測試
quick_mcp_test() {
    echo -n "測試 MCP Server... "
    
    # 檢查 MCP Server 是否存在
    if [ ! -f "bin/mcp-server" ]; then
        echo -e "${YELLOW}⚠️  需要建構${NC}"
        echo "建構 MCP Server..."
        go build -o bin/mcp-server ./cmd/mcp-server
    fi
    
    local response=$(echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | timeout 5 ./bin/mcp-server 2>/dev/null || echo "TIMEOUT")
    
    if [ "$response" != "TIMEOUT" ] && echo "$response" | jq -e '.result' >/dev/null 2>&1; then
        echo -e "${GREEN}✅ 通過${NC}"
        return 0
    else
        echo -e "${RED}❌ 失敗${NC}"
        return 1
    fi
}

# 主函數
main() {
    local failed=0
    
    check_service
    
    echo "執行快速測試..."
    
    quick_api_test || ((failed++))
    quick_upload_test || ((failed++))
    quick_search_test || ((failed++))
    quick_mcp_test || ((failed++))
    
    echo ""
    if [ $failed -eq 0 ]; then
        echo -e "${GREEN}🎉 所有快速測試通過！${NC}"
        echo "系統基本功能正常，可以進行完整測試"
    else
        echo -e "${RED}❌ $failed 個測試失敗${NC}"
        echo "請檢查系統配置和服務狀態"
        exit 1
    fi
}

# 執行主函數
main "$@"