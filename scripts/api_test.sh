#!/bin/bash
# API 測試腳本

set -e

BASE_URL="http://localhost:8080"
API_KEY="${API_KEY:-test-api-key}"

echo "🧪 開始 API 測試..."

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 測試函數
test_api() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local data="$4"
    local expected_status="${5:-200}"
    
    echo -n "測試 $name... "
    
    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "%{http_code}" -H "Authorization: Bearer $API_KEY" "$BASE_URL$endpoint")
    else
        response=$(curl -s -w "%{http_code}" -X "$method" -H "Content-Type: application/json" -H "Authorization: Bearer $API_KEY" -d "$data" "$BASE_URL$endpoint")
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

# 1. 健康檢查
test_api "健康檢查" "GET" "/health"

# 2. 測試基本 API 端點
test_api "取得 chunks" "GET" "/api/v1/chunks"

# 3. 測試圖片上傳 API（使用 base64 編碼的小圖片）
# 建立一個簡單的 1x1 像素 PNG 圖片的 base64
SMALL_PNG_BASE64="iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="

test_api "圖片上傳" "POST" "/api/v1/media/upload" '{
    "image_data": "'$SMALL_PNG_BASE64'",
    "filename": "test.png",
    "auto_analyze": true,
    "auto_embed": true,
    "storage_type": "supabase"
}'

# 4. 測試多模態搜尋
test_api "多模態搜尋" "POST" "/api/v1/search/multimodal" '{
    "text_query": "測試",
    "search_type": "hybrid",
    "limit": 10,
    "min_similarity": 0.7
}'

# 5. 測試圖片分析
test_api "圖片分析" "POST" "/api/v1/media/analyze" '{
    "image_url": "https://via.placeholder.com/150",
    "detail_level": "medium",
    "language": "zh-TW"
}'

# 6. 測試批次處理狀態
test_api "批次處理狀態" "GET" "/api/v1/media/batch/status"

# 7. 測試投影片推薦
test_api "投影片推薦" "POST" "/api/v1/media/recommend-slides" '{
    "slide_title": "測試投影片",
    "slide_content": "這是一個測試投影片的內容",
    "max_suggestions": 5,
    "min_relevance": 0.6
}'

# 8. 測試重複圖片檢測
test_api "重複圖片檢測" "POST" "/api/v1/media/find-duplicates" '{
    "similarity_threshold": 0.95,
    "min_group_size": 2
}'

# 9. 測試圖片庫
test_api "圖片庫" "GET" "/api/v1/media/library?page=1&limit=10"

echo ""
echo -e "${GREEN}🎉 API 測試完成！${NC}"