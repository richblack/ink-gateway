#!/bin/bash
# Docker 環境整合測試腳本

set -e

echo "🐳 Docker 環境整合測試開始"

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

GATEWAY_URL="${GATEWAY_URL:-http://ink-gateway:8080}"
TEST_RESULTS=()

# 等待服務就緒
wait_for_service() {
    local service_url="$1"
    local service_name="$2"
    local max_attempts=30
    
    echo -n "等待 $service_name 服務就緒"
    for i in $(seq 1 $max_attempts); do
        if curl -s "$service_url/health" >/dev/null 2>&1; then
            echo -e "\n${GREEN}✅ $service_name 服務就緒${NC}"
            return 0
        fi
        echo -n "."
        sleep 2
    done
    
    echo -e "\n${RED}❌ $service_name 服務未就緒${NC}"
    return 1
}

# 測試 API 端點
test_api_endpoint() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local data="$4"
    local expected_status="${5:-200}"
    
    echo -n "測試 $name... "
    
    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "%{http_code}" "$GATEWAY_URL$endpoint")
    else
        response=$(curl -s -w "%{http_code}" -X "$method" -H "Content-Type: application/json" -d "$data" "$GATEWAY_URL$endpoint")
    fi
    
    status_code="${response: -3}"
    body="${response%???}"
    
    if [ "$status_code" = "$expected_status" ]; then
        echo -e "${GREEN}✅ 通過${NC}"
        return 0
    else
        echo -e "${RED}❌ 失敗 (狀態碼: $status_code)${NC}"
        echo "回應: $body"
        return 1
    fi
}

# 執行基本 API 測試
run_basic_api_tests() {
    echo -e "${BLUE}🌐 執行基本 API 測試...${NC}"
    
    local success=true
    
    # 健康檢查
    if test_api_endpoint "健康檢查" "GET" "/health"; then
        TEST_RESULTS+=("健康檢查: ✅ 通過")
    else
        TEST_RESULTS+=("健康檢查: ❌ 失敗")
        success=false
    fi
    
    # 基本端點測試
    if test_api_endpoint "取得 chunks" "GET" "/api/v1/chunks"; then
        TEST_RESULTS+=("基本端點: ✅ 通過")
    else
        TEST_RESULTS+=("基本端點: ❌ 失敗")
        success=false
    fi
    
    # 圖片上傳測試
    local png_data="iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="
    if test_api_endpoint "圖片上傳" "POST" "/api/v1/media/upload" "{\"image_data\":\"$png_data\",\"filename\":\"docker_test.png\"}"; then
        TEST_RESULTS+=("圖片上傳: ✅ 通過")
    else
        TEST_RESULTS+=("圖片上傳: ❌ 失敗")
        success=false
    fi
    
    if $success; then
        echo -e "${GREEN}✅ 基本 API 測試通過${NC}\n"
    else
        echo -e "${RED}❌ 基本 API 測試失敗${NC}\n"
    fi
}

# 執行多模態搜尋測試
run_multimodal_search_tests() {
    echo -e "${BLUE}🔍 執行多模態搜尋測試...${NC}"
    
    local success=true
    
    # 文字搜尋
    if test_api_endpoint "文字搜尋" "POST" "/api/v1/search/multimodal" '{"text_query":"測試","search_type":"text","limit":5}'; then
        TEST_RESULTS+=("文字搜尋: ✅ 通過")
    else
        TEST_RESULTS+=("文字搜尋: ❌ 失敗")
        success=false
    fi
    
    # 混合搜尋
    if test_api_endpoint "混合搜尋" "POST" "/api/v1/search/multimodal" '{"text_query":"測試","search_type":"hybrid","limit":5}'; then
        TEST_RESULTS+=("混合搜尋: ✅ 通過")
    else
        TEST_RESULTS+=("混合搜尋: ❌ 失敗")
        success=false
    fi
    
    if $success; then
        echo -e "${GREEN}✅ 多模態搜尋測試通過${NC}\n"
    else
        echo -e "${RED}❌ 多模態搜尋測試失敗${NC}\n"
    fi
}

# 執行負載測試
run_load_tests() {
    echo -e "${BLUE}⚡ 執行負載測試...${NC}"
    
    local start_time=$(date +%s)
    local png_data="iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="
    
    # 並發上傳測試
    for i in {1..10}; do
        curl -s -X POST "$GATEWAY_URL/api/v1/media/upload" \
            -H "Content-Type: application/json" \
            -d "{\"image_data\":\"$png_data\",\"filename\":\"load_test_$i.png\"}" &
    done
    
    wait
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo "並發上傳 10 張圖片耗時: ${duration}s"
    
    if [ $duration -lt 60 ]; then
        TEST_RESULTS+=("負載測試: ✅ 通過 (${duration}s)")
        echo -e "${GREEN}✅ 負載測試通過${NC}\n"
    else
        TEST_RESULTS+=("負載測試: ⚠️  較慢 (${duration}s)")
        echo -e "${YELLOW}⚠️  負載測試較慢${NC}\n"
    fi
}

# 生成測試報告
generate_docker_report() {
    echo -e "${BLUE}📊 生成 Docker 測試報告...${NC}"
    
    local report_file="/app/test-results/docker_test_report_$(date +%Y%m%d_%H%M%S).md"
    
    cat > "$report_file" << EOF
# Docker 環境整合測試報告

**測試時間**: $(date)
**測試環境**: Docker 容器
**Gateway URL**: $GATEWAY_URL

## 測試結果

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

## 容器資訊

- **主機名**: $(hostname)
- **容器 ID**: $(hostname)
- **記憶體使用**: $(free -h | grep Mem | awk '{print $3 "/" $2}' 2>/dev/null || echo "N/A")

EOF
    
    echo -e "${GREEN}✅ Docker 測試報告已生成: $report_file${NC}"
    
    # 顯示摘要
    echo -e "\n${BLUE}📈 Docker 測試摘要:${NC}"
    for result in "${TEST_RESULTS[@]}"; do
        echo "  $result"
    done
    
    echo -e "\n${GREEN}通過率: $(( passed * 100 / total ))% ($passed/$total)${NC}"
    
    # 複製報告到共享卷
    cp "$report_file" /app/test-results/latest_docker_report.md
}

# 主函數
main() {
    echo -e "${GREEN}🎯 Docker 環境整合測試${NC}"
    echo "=================================="
    
    # 等待服務就緒
    wait_for_service "$GATEWAY_URL" "Ink-Gateway"
    
    # 執行測試
    run_basic_api_tests
    run_multimodal_search_tests
    run_load_tests
    
    # 生成報告
    generate_docker_report
    
    echo -e "\n${GREEN}🎉 Docker 整合測試完成！${NC}"
    
    # 返回適當的退出碼
    local failed_tests=0
    for result in "${TEST_RESULTS[@]}"; do
        if [[ $result == *"❌"* ]]; then
            ((failed_tests++))
        fi
    done
    
    if [ $failed_tests -eq 0 ]; then
        exit 0
    else
        exit 1
    fi
}

# 執行主函數
main "$@"