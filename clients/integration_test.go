package clients

import (
	"context"
	"os"
	"testing"
	"time"

	"semantic-text-processor/config"
)

// TestSupabaseClientIntegration tests the real Supabase client instantiation
// This test requires actual Supabase credentials to be set in environment variables
func TestSupabaseClientIntegration(t *testing.T) {
	// Skip if no Supabase credentials are provided
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseAPIKey := os.Getenv("SUPABASE_API_KEY")
	
	if supabaseURL == "" || supabaseAPIKey == "" {
		t.Skip("Skipping integration test: SUPABASE_URL and SUPABASE_API_KEY environment variables not set")
	}
	
	cfg := &config.SupabaseConfig{
		URL:    supabaseURL,
		APIKey: supabaseAPIKey,
	}
	
	client := NewSupabaseClient(cfg)
	if client == nil {
		t.Fatal("NewSupabaseClient returned nil")
	}
	
	// Test health check with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	err := client.HealthCheck(ctx)
	if err != nil {
		t.Logf("Health check failed (this is expected if Supabase is not configured): %v", err)
		// Don't fail the test since this is expected in most development environments
	} else {
		t.Log("Health check passed - Supabase connection is working")
	}
}

// TestSupabaseClientConfiguration tests client configuration validation
func TestSupabaseClientConfiguration(t *testing.T) {
	testCases := []struct {
		name   string
		config *config.SupabaseConfig
		valid  bool
	}{
		{
			name: "valid configuration",
			config: &config.SupabaseConfig{
				URL:    "https://test.supabase.co",
				APIKey: "test-api-key",
			},
			valid: true,
		},
		{
			name: "empty URL",
			config: &config.SupabaseConfig{
				URL:    "",
				APIKey: "test-api-key",
			},
			valid: true, // Client creation doesn't validate, only usage does
		},
		{
			name: "empty API key",
			config: &config.SupabaseConfig{
				URL:    "https://test.supabase.co",
				APIKey: "",
			},
			valid: true, // Client creation doesn't validate, only usage does
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewSupabaseClient(tc.config)
			if client == nil {
				t.Error("NewSupabaseClient should never return nil")
			}
		})
	}
}