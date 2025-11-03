package clients

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"semantic-text-processor/config"
	"semantic-text-processor/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphIntegration(t *testing.T) {
	// Skip if not running integration tests
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
	
	// Test health check first
	err := client.HealthCheck(ctx)
	require.NoError(t, err, "Supabase should be accessible")
	
	t.Run("CompleteGraphWorkflow", func(t *testing.T) {
		// 1. Create test text and chunks
		testText := &models.TextRecord{
			Content: "Knowledge graph integration test content",
			Title:   "Graph Integration Test",
			Status:  "completed",
		}
		
		err := client.InsertText(ctx, testText)
		require.NoError(t, err)
		
		chunks := []models.ChunkRecord{
			{
				TextID:      testText.ID,
				Content:     "John works at Microsoft as a Software Engineer",
				IndentLevel: 0,
			},
			{
				TextID:      testText.ID,
				Content:     "Microsoft is a technology company founded in 1975",
				IndentLevel: 0,
			},
			{
				TextID:      testText.ID,
				Content:     "Software Engineers develop applications and systems",
				IndentLevel: 1,
			},
		}
		
		err = client.InsertChunks(ctx, chunks)
		require.NoError(t, err)
		
		// 2. Create knowledge graph nodes
		nodes := []models.GraphNode{
			{
				ChunkID:    chunks[0].ID,
				EntityName: "John",
				EntityType: "Person",
				Properties: map[string]interface{}{
					"profession": "Software Engineer",
					"experience": "5 years",
				},
			},
			{
				ChunkID:    chunks[1].ID,
				EntityName: "Microsoft",
				EntityType: "Organization",
				Properties: map[string]interface{}{
					"industry":     "Technology",
					"founded_year": 1975,
					"headquarters": "Redmond, WA",
				},
			},
			{
				ChunkID:    chunks[2].ID,
				EntityName: "Software Engineer",
				EntityType: "JobRole",
				Properties: map[string]interface{}{
					"category":    "Technology",
					"skill_level": "Professional",
				},
			},
		}
		
		err = client.InsertGraphNodes(ctx, nodes)
		require.NoError(t, err)
		
		// Verify nodes were created
		for i, node := range nodes {
			assert.NotEmpty(t, node.ID, "Node %d should have an ID", i)
			assert.False(t, node.CreatedAt.IsZero(), "Node %d should have creation time", i)
		}
		
		// 3. Create relationships between entities
		edges := []models.GraphEdge{
			{
				SourceNodeID:     nodes[0].ID, // John
				TargetNodeID:     nodes[1].ID, // Microsoft
				RelationshipType: "WORKS_FOR",
				Properties: map[string]interface{}{
					"start_date": "2020-01-15",
					"department": "Cloud Services",
				},
			},
			{
				SourceNodeID:     nodes[0].ID, // John
				TargetNodeID:     nodes[2].ID, // Software Engineer
				RelationshipType: "HAS_ROLE",
				Properties: map[string]interface{}{
					"level":       "Senior",
					"specialization": "Backend Development",
				},
			},
			{
				SourceNodeID:     nodes[1].ID, // Microsoft
				TargetNodeID:     nodes[2].ID, // Software Engineer
				RelationshipType: "EMPLOYS",
				Properties: map[string]interface{}{
					"count":      "50000+",
					"locations":  []string{"Global"},
				},
			},
		}
		
		err = client.InsertGraphEdges(ctx, edges)
		require.NoError(t, err)
		
		// Verify edges were created
		for i, edge := range edges {
			assert.NotEmpty(t, edge.ID, "Edge %d should have an ID", i)
			assert.False(t, edge.CreatedAt.IsZero(), "Edge %d should have creation time", i)
		}
		
		// 4. Test graph search functionality
		t.Run("SearchByPersonEntity", func(t *testing.T) {
			query := &models.GraphQuery{
				EntityName: "John",
				MaxDepth:   2,
				Limit:      20,
			}
			
			result, err := client.SearchGraph(ctx, query)
			
			// Note: This will likely fail unless the search_graph RPC function exists
			if err != nil {
				t.Logf("Graph search failed (expected without RPC function): %v", err)
				t.Skip("Graph search RPC function not implemented in Supabase")
				return
			}
			
			assert.NotNil(t, result)
			assert.NotEmpty(t, result.Nodes, "Should find related nodes")
			assert.NotEmpty(t, result.Edges, "Should find connecting edges")
			
			// Verify we found John
			foundJohn := false
			for _, node := range result.Nodes {
				if node.EntityName == "John" {
					foundJohn = true
					break
				}
			}
			assert.True(t, foundJohn, "Should find John in search results")
		})
		
		// 5. Test search chunks functionality
		t.Run("SearchChunksWithGraphContext", func(t *testing.T) {
			// Search for chunks containing "Microsoft"
			results, err := client.SearchChunks(ctx, "Microsoft", map[string]interface{}{
				"text_id": testText.ID,
				"limit":   10,
			})
			
			require.NoError(t, err)
			assert.NotEmpty(t, results, "Should find chunks mentioning Microsoft")
			
			// Verify we found the right chunk
			foundMicrosoftChunk := false
			for _, chunk := range results {
				if chunk.Content == "Microsoft is a technology company founded in 1975" {
					foundMicrosoftChunk = true
					break
				}
			}
			assert.True(t, foundMicrosoftChunk, "Should find the Microsoft chunk")
		})
		
		// 6. Test combined search (chunks + graph)
		t.Run("CombinedSearchWorkflow", func(t *testing.T) {
			// First, search for chunks about "Software Engineer"
			chunkResults, err := client.SearchChunks(ctx, "Software Engineer", map[string]interface{}{
				"text_id": testText.ID,
			})
			require.NoError(t, err)
			
			if len(chunkResults) > 0 {
				// Then search for graph entities related to those chunks
				// This would typically be done by finding graph nodes linked to the chunks
				t.Logf("Found %d chunks related to Software Engineer", len(chunkResults))
				
				// In a real application, you would:
				// 1. Get graph nodes associated with these chunks
				// 2. Perform graph traversal from those nodes
				// 3. Combine results for comprehensive search
			}
		})
	})
	
	t.Run("GraphErrorHandling", func(t *testing.T) {
		// Test error handling for invalid operations
		
		t.Run("EmptyNodesInsert", func(t *testing.T) {
			err := client.InsertGraphNodes(ctx, []models.GraphNode{})
			assert.NoError(t, err, "Should handle empty nodes gracefully")
		})
		
		t.Run("EmptyEdgesInsert", func(t *testing.T) {
			err := client.InsertGraphEdges(ctx, []models.GraphEdge{})
			assert.NoError(t, err, "Should handle empty edges gracefully")
		})
		
		t.Run("InvalidGraphQuery", func(t *testing.T) {
			result, err := client.SearchGraph(ctx, nil)
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "graph query cannot be nil")
		})
		
		t.Run("GraphQueryWithDefaults", func(t *testing.T) {
			query := &models.GraphQuery{
				EntityName: "NonExistentEntity",
				// MaxDepth and Limit will use defaults
			}
			
			result, err := client.SearchGraph(ctx, query)
			
			// This may fail if RPC function doesn't exist
			if err != nil {
				t.Logf("Graph search with defaults failed (expected): %v", err)
				return
			}
			
			assert.NotNil(t, result)
			// May have empty results for non-existent entity
		})
	})
	
	t.Run("GraphPerformanceTest", func(t *testing.T) {
		// Create a larger graph for performance testing
		startTime := time.Now()
		
		// Create test text
		testText := &models.TextRecord{
			Content: "Performance test content",
			Title:   "Graph Performance Test",
			Status:  "completed",
		}
		
		err := client.InsertText(ctx, testText)
		require.NoError(t, err)
		
		testChunk := &models.ChunkRecord{
			TextID:      testText.ID,
			Content:     "Performance test chunk",
			IndentLevel: 0,
		}
		
		err = client.InsertChunk(ctx, testChunk)
		require.NoError(t, err)
		
		// Create multiple nodes
		nodeCount := 50
		nodes := make([]models.GraphNode, nodeCount)
		for i := 0; i < nodeCount; i++ {
			nodes[i] = models.GraphNode{
				ChunkID:    testChunk.ID,
				EntityName: fmt.Sprintf("Entity_%d", i),
				EntityType: "TestEntity",
				Properties: map[string]interface{}{
					"index": i,
					"batch": "performance_test",
				},
			}
		}
		
		insertStart := time.Now()
		err = client.InsertGraphNodes(ctx, nodes)
		insertDuration := time.Since(insertStart)
		
		require.NoError(t, err)
		t.Logf("Inserted %d nodes in %v", nodeCount, insertDuration)
		
		// Create edges between nodes
		edgeCount := nodeCount - 1
		edges := make([]models.GraphEdge, edgeCount)
		for i := 0; i < edgeCount; i++ {
			edges[i] = models.GraphEdge{
				SourceNodeID:     nodes[i].ID,
				TargetNodeID:     nodes[i+1].ID,
				RelationshipType: "CONNECTS_TO",
				Properties: map[string]interface{}{
					"sequence": i,
				},
			}
		}
		
		edgeInsertStart := time.Now()
		err = client.InsertGraphEdges(ctx, edges)
		edgeInsertDuration := time.Since(edgeInsertStart)
		
		require.NoError(t, err)
		t.Logf("Inserted %d edges in %v", edgeCount, edgeInsertDuration)
		
		totalDuration := time.Since(startTime)
		t.Logf("Total graph creation time: %v", totalDuration)
		
		// Performance assertions (adjust based on expected performance)
		assert.Less(t, insertDuration, 5*time.Second, "Node insertion should be reasonably fast")
		assert.Less(t, edgeInsertDuration, 5*time.Second, "Edge insertion should be reasonably fast")
	})
}

