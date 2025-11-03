package clients

import (
	"context"
	"os"
	"testing"

	"semantic-text-processor/config"

	"github.com/stretchr/testify/assert"
)

func TestSupabaseConnection(t *testing.T) {
	// 只在設置了環境變數時運行
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests. Set INTEGRATION_TESTS=true to run.")
	}

	cfg := &config.SupabaseConfig{
		URL:    os.Getenv("SUPABASE_URL"),
		APIKey: os.Getenv("SUPABASE_API_KEY"),
	}

	if cfg.URL == "" || cfg.APIKey == "" {
		t.Skip("Supabase configuration not provided via environment variables")
	}

	client := NewSupabaseClient(cfg)
	ctx := context.Background()

	t.Run("BasicConnection", func(t *testing.T) {
		// 測試基本的 HTTP 連接
		httpClient := client.(*supabaseHTTPClient)
		
		// 嘗試一個簡單的請求來測試連接
		var result []map[string]interface{}
		err := httpClient.makeRequest(ctx, "GET", "/", nil, &result)
		
		// 我們期望得到某種回應，即使是錯誤也表示連接成功
		t.Logf("Connection test result: %v", err)
		
		// 只要不是網路連接錯誤就算成功
		if err != nil {
			assert.NotContains(t, err.Error(), "connection refused")
			assert.NotContains(t, err.Error(), "no such host")
		}
	})

	t.Run("APIKeyValidation", func(t *testing.T) {
		// 測試 API key 是否有效
		httpClient := client.(*supabaseHTTPClient)
		
		// 嘗試訪問一個需要認證的端點
		var result []map[string]interface{}
		err := httpClient.makeRequest(ctx, "GET", "/auth/v1/user", nil, &result)
		
		t.Logf("API key validation result: %v", err)
		
		// 只要不是認證錯誤就算 API key 有效
		if err != nil {
			assert.NotContains(t, err.Error(), "Invalid API key")
			assert.NotContains(t, err.Error(), "unauthorized")
		}
	})
}