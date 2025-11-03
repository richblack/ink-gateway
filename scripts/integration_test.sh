#!/bin/bash
# 完整整合測試腳本

set -e

echo "🚀 開始多模態 MCP 系統整合測試"

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 全域變數
GATEWAY_PID=""
TEST_RESULTS=()

# 清理函數
cleanup() {
    echo -e "\n${YELLOW}🧹 清理資源...${NC}"
    if [ -n "$GATEWAY_PID" ]; then
        kill $GATEWAY_PID 2>/dev/null || true
        echo "已停止 Ink-Gateway (PID: $GATEWAY_PID)"
    fi
    
    # 清理測試檔案
    rm -rf test-images test-batch 2>/dev/null || true
    
    echo -e "${GREEN}清理完成${NC}"
}

# 設定 trap 來確保清理
trap cleanup EXIT

# 檢查依賴
check_dependencies() {
    echo -e "${BLUE}📋 檢查環境依賴...${NC}"
    
    local deps=("go" "node" "curl" "jq")
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" >/dev/null 2>&1; then
            echo -e "${RED}❌ 缺少依賴: $dep${NC}"
            exit 1
        fi
        echo -e "${GREEN}✅ $dep${NC}"
    done
    
    # 檢查環境變數
    if [ -z "$SUPABASE_URL" ] || [ -z "$SUPABASE_API_KEY" ]; then
        echo -e "${YELLOW}⚠️  警告: 缺少 Supabase 環境變數，某些測試可能失敗${NC}"
    fi
    
    echo -e "${GREEN}環境檢查完成${NC}\n"
}

# 啟動 Ink-Gateway
start_gateway() {
    echo -e "${BLUE}🔧 啟動 Ink-Gateway...${NC}"
    
    # 檢查端口是否被占用
    if lsof -i :8080 >/dev/null 2>&1; then
        echo -e "${YELLOW}⚠️  端口 8080 已被占用，嘗試停止現有服務...${NC}"
        pkill -f "ink-gateway" 2>/dev/null || true
        sleep 2
    fi
    
    # 啟動服務
    make run > gateway.log 2>&1 &
    GATEWAY_PID=$!
    
    echo "Ink-Gateway PID: $GATEWAY_PID"
    
    # 等待服務啟動
    echo -n "等待服務啟動"
    for i in {1..30}; do
        if curl -s http://localhost:8080/health >/dev/null 2>&1; then
            echo -e "\n${GREEN}✅ Ink-Gateway 啟動成功${NC}"
            return 0
        fi
        echo -n "."
        sleep 1
    done
    
    echo -e "\n${RED}❌ Ink-Gateway 啟動失敗${NC}"
    echo "日誌內容:"
    tail -20 gateway.log
    exit 1
}

# 建構組件
build_components() {
    echo -e "${BLUE}🔨 建構組件...${NC}"
    
    # 建構 MCP Server
    echo "建構 MCP Server..."
    if go build -o bin/mcp-server ./cmd/mcp-server; then
        echo -e "${GREEN}✅ MCP Server 建構成功${NC}"
    else
        echo -e "${RED}❌ MCP Server 建構失敗${NC}"
        exit 1
    fi
    
    # 建構 Obsidian 插件
    echo "建構 Obsidian 插件..."
    cd obsidian-ink-plugin
    if npm install && npm run build; then
        echo -e "${GREEN}✅ Obsidian 插件建構成功${NC}"
    else
        echo -e "${RED}❌ Obsidian 插件建構失敗${NC}"
        exit 1
    fi
    cd ..
    
    echo -e "${GREEN}組件建構完成${NC}\n"
}

# 準備測試資料
prepare_test_data() {
    echo -e "${BLUE}📁 準備測試資料...${NC}"
    
    # 建立測試圖片目錄
    mkdir -p test-images test-batch
    
    # 建立簡單的測試圖片（1x1 像素 PNG）
    local png_data="iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="
    
    for i in {1..5}; do
        echo "$png_data" | base64 -d > "test-images/test$i.png"
        echo "$png_data" | base64 -d > "test-batch/batch$i.png"
    done
    
    echo -e "${GREEN}✅ 測試資料準備完成${NC}\n"
}

# 執行 API 測試
run_api_tests() {
    echo -e "${BLUE}🌐 執行 API 測試...${NC}"
    
    if bash scripts/api_test.sh; then
        TEST_RESULTS+=("API測試: ✅ 通過")
        echo -e "${GREEN}✅ API 測試通過${NC}\n"
    else
        TEST_RESULTS+=("API測試: ❌ 失敗")
        echo -e "${RED}❌ API 測試失敗${NC}\n"
    fi
}

# 執行 MCP 測試
run_mcp_tests() {
    echo -e "${BLUE}🔧 執行 MCP Server 測試...${NC}"
    
    if bash scripts/mcp_test.sh; then
        TEST_RESULTS+=("MCP測試: ✅ 通過")
        echo -e "${GREEN}✅ MCP Server 測試通過${NC}\n"
    else
        TEST_RESULTS+=("MCP測試: ❌ 失敗")
        echo -e "${RED}❌ MCP Server 測試失敗${NC}\n"
    fi
}

# 執行端到端測試
run_e2e_tests() {
    echo -e "${BLUE}🔄 執行端到端測試...${NC}"
    
    local success=true
    
    # 測試 1: 圖片上傳流程
    echo "測試圖片上傳流程..."
    local upload_response=$(curl -s -X POST http://localhost:8080/api/v1/media/upload \
        -H "Content-Type: application/json" \
        -d '{
            "image_data": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==",
            "filename": "e2e_test.png",
            "auto_analyze": true,
            "auto_embed": true
        }')
    
    if echo "$upload_response" | jq -e '.chunk_id' >/dev/null 2>&1; then
        echo -e "${GREEN}✅ 圖片上傳測試通過${NC}"
        local chunk_id=$(echo "$upload_response" | jq -r '.chunk_id')
        
        # 測試 2: 使用上傳的圖片進行搜尋
        echo "測試多模態搜尋..."
        local search_response=$(curl -s -X POST http://localhost:8080/api/v1/search/multimodal \
            -H "Content-Type: application/json" \
            -d '{
                "text_query": "test",
                "search_type": "hybrid",
                "limit": 5
            }')
        
        if echo "$search_response" | jq -e '.results' >/dev/null 2>&1; then
            echo -e "${GREEN}✅ 多模態搜尋測試通過${NC}"
        else
            echo -e "${RED}❌ 多模態搜尋測試失敗${NC}"
            success=false
        fi
        
    else
        echo -e "${RED}❌ 圖片上傳測試失敗${NC}"
        success=false
    fi
    
    # 測試 3: MCP 工具整合
    echo "測試 MCP 工具整合..."
    local mcp_response=$(echo '{
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/call",
        "params": {
            "name": "ink_search_chunks",
            "arguments": {
                "query": "test",
                "limit": 3
            }
        }
    }' | timeout 15 ./bin/mcp-server 2>/dev/null || echo "TIMEOUT")
    
    if [ "$mcp_response" != "TIMEOUT" ] && echo "$mcp_response" | jq -e '.result' >/dev/null 2>&1; then
        echo -e "${GREEN}✅ MCP 工具整合測試通過${NC}"
    else
        echo -e "${RED}❌ MCP 工具整合測試失敗${NC}"
        success=false
    fi
    
    if $success; then
        TEST_RESULTS+=("端到端測試: ✅ 通過")
        echo -e "${GREEN}✅ 端到端測試通過${NC}\n"
    else
        TEST_RESULTS+=("端到端測試: ❌ 失敗")
        echo -e "${RED}❌ 端到端測試失敗${NC}\n"
    fi
}

# 效能測試
run_performance_tests() {
    echo -e "${BLUE}⚡ 執行效能測試...${NC}"
    
    # 測試並發上傳
    echo "測試並發圖片上傳..."
    local start_time=$(date +%s)
    
    for i in {1..5}; do
        curl -s -X POST http://localhost:8080/api/v1/media/upload \
            -H "Content-Type: application/json" \
            -d '{
                "image_data": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==",
                "filename": "perf_test_'$i'.png"
            }' &
    done
    
    wait
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo "並發上傳 5 張圖片耗時: ${duration}s"
    
    if [ $duration -lt 30 ]; then
        TEST_RESULTS+=("效能測試: ✅ 通過 (${duration}s)")
        echo -e "${GREEN}✅ 效能測試通過${NC}\n"
    else
        TEST_RESULTS+=("效能測試: ⚠️  較慢 (${duration}s)")
        echo -e "${YELLOW}⚠️  效能測試較慢${NC}\n"
    fi
}

# 生成測試報告
generate_report() {
    echo -e "${BLUE}📊 生成測試報告...${NC}"
    
    local report_file="test_report_$(date +%Y%m%d_%H%M%S).md"
    
    cat > "$report_file" << EOF
# 多模態 MCP 系統整合測試報告

**測試時間**: $(date)
**測試環境**: $(uname -a)

## 測試結果摘要

EOF
    
    local passed=0
    local total=${#TEST_RESULTS[@]}
    
    for result in "${TEST_RESULTS[@]}"; do
        echo "- $result" >> "$report_file"
        if [[ $result == *"✅"* ]]; then
            ((passed++))
        fi
    done
    
    cat >> "$report_file" << EOF

## 統計

- **總測試項目**: $total
- **通過項目**: $passed
- **失敗項目**: $((total - passed))
- **通過率**: $(( passed * 100 / total ))%

## 系統資訊

- **Go 版本**: $(go version)
- **Node.js 版本**: $(node --version)
- **系統記憶體**: $(free -h | grep Mem | awk '{print $2}' 2>/dev/null || echo "N/A")
- **磁碟空間**: $(df -h . | tail -1 | awk '{print $4}' 2>/dev/null || echo "N/A")

## 日誌檔案

- Gateway 日誌: gateway.log
- MCP Server 日誌: mcp-server.log (如果有)

EOF
    
    echo -e "${GREEN}✅ 測試報告已生成: $report_file${NC}"
    
    # 顯示摘要
    echo -e "\n${BLUE}📈 測試摘要:${NC}"
    for result in "${TEST_RESULTS[@]}"; do
        echo "  $result"
    done
    
    echo -e "\n${GREEN}通過率: $(( passed * 100 / total ))% ($passed/$total)${NC}"
}

# 主函數
main() {
    echo -e "${GREEN}🎯 多模態 MCP 系統整合測試${NC}"
    echo "=================================="
    
    check_dependencies
    prepare_test_data
    build_components
    start_gateway
    
    run_api_tests
    run_mcp_tests
    run_e2e_tests
    run_performance_tests
    
    generate_report
    
    echo -e "\n${GREEN}🎉 整合測試完成！${NC}"
}

# 執行主函數
main "$@"