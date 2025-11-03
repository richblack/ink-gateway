package clients

import (
	"context"
	"os"
	"testing"

	"semantic-text-processor/config"
	"semantic-text-processor/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableVerification(t *testing.T) {
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

	t.Run("VerifyTextsTable", func(t *testing.T) {
		testText := &models.TextRecord{
			Content: "Test content for table verification",
			Title:   "Table Verification Test",
			Status:  "completed",
		}

		err := client.InsertText(ctx, testText)
		if err != nil {
			t.Logf("❌ texts table not ready: %v", err)
			t.Fail()
			return
		}

		t.Log("✅ texts table is working")
		require.NotEmpty(t, testText.ID)
		assert.False(t, testText.CreatedAt.IsZero())

		// 測試檢索
		retrievedText, err := client.GetTextByID(ctx, testText.ID)
		require.NoError(t, err)
		assert.Equal(t, testText.Content, retrievedText.Text.Content)
		t.Log("✅ texts table read operation working")

		// 清理
		err = client.DeleteText(ctx, testText.ID)
		assert.NoError(t, err)
		t.Log("✅ texts table delete operation working")
	})

	t.Run("VerifyChunksTable", func(t *testing.T) {
		// 先創建一個 text
		testText := &models.TextRecord{
			Content: "Parent text for chunks",
			Title:   "Chunks Test",
			Status:  "completed",
		}
		err := client.InsertText(ctx, testText)
		require.NoError(t, err)

		// 創建 chunk
		testChunk := &models.ChunkRecord{
			TextID:      testText.ID,
			Content:     "Test chunk content",
			IndentLevel: 0,
		}

		err = client.InsertChunk(ctx, testChunk)
		if err != nil {
			t.Logf("❌ chunks table not ready: %v", err)
			client.DeleteText(ctx, testText.ID) // 清理
			t.Fail()
			return
		}

		t.Log("✅ chunks table is working")
		require.NotEmpty(t, testChunk.ID)

		// 測試檢索
		retrievedChunk, err := client.GetChunkByID(ctx, testChunk.ID)
		require.NoError(t, err)
		assert.Equal(t, testChunk.Content, retrievedChunk.Content)
		t.Log("✅ chunks table read operation working")

		// 清理
		client.DeleteText(ctx, testText.ID) // 這會級聯刪除 chunks
	})

	t.Run("VerifyGraphNodesTable", func(t *testing.T) {
		// 先創建必要的 text 和 chunk
		testText := &models.TextRecord{
			Content: "Text for graph test",
			Title:   "Graph Test",
			Status:  "completed",
		}
		err := client.InsertText(ctx, testText)
		require.NoError(t, err)

		testChunk := &models.ChunkRecord{
			TextID:      testText.ID,
			Content:     "Chunk for graph node",
			IndentLevel: 0,
		}
		err = client.InsertChunk(ctx, testChunk)
		require.NoError(t, err)

		// 創建 graph node
		nodes := []models.GraphNode{
			{
				ChunkID:    testChunk.ID,
				EntityName: "Test Entity",
				EntityType: "TestType",
				Properties: map[string]interface{}{
					"test": "value",
				},
			},
		}

		err = client.InsertGraphNodes(ctx, nodes)
		if err != nil {
			t.Logf("❌ graph_nodes table not ready: %v", err)
			client.DeleteText(ctx, testText.ID) // 清理
			t.Fail()
			return
		}

		t.Log("✅ graph_nodes table is working")
		require.NotEmpty(t, nodes[0].ID)

		// 測試檢索
		foundNodes, err := client.GetNodesByEntity(ctx, "Test Entity")
		require.NoError(t, err)
		assert.Len(t, foundNodes, 1)
		assert.Equal(t, "Test Entity", foundNodes[0].EntityName)
		t.Log("✅ graph_nodes table read operation working")

		// 清理
		client.DeleteText(ctx, testText.ID) // 這會級聯刪除相關數據
	})

	t.Run("VerifyGraphEdgesTable", func(t *testing.T) {
		// 先創建必要的數據
		testText := &models.TextRecord{
			Content: "Text for graph edges test",
			Title:   "Graph Edges Test",
			Status:  "completed",
		}
		err := client.InsertText(ctx, testText)
		require.NoError(t, err)

		testChunk := &models.ChunkRecord{
			TextID:      testText.ID,
			Content:     "Chunk for graph edges",
			IndentLevel: 0,
		}
		err = client.InsertChunk(ctx, testChunk)
		require.NoError(t, err)

		// 創建兩個 nodes
		nodes := []models.GraphNode{
			{
				ChunkID:    testChunk.ID,
				EntityName: "Source Entity",
				EntityType: "TestType",
			},
			{
				ChunkID:    testChunk.ID,
				EntityName: "Target Entity",
				EntityType: "TestType",
			},
		}
		err = client.InsertGraphNodes(ctx, nodes)
		require.NoError(t, err)

		// 創建 edge
		edges := []models.GraphEdge{
			{
				SourceNodeID:     nodes[0].ID,
				TargetNodeID:     nodes[1].ID,
				RelationshipType: "TEST_RELATION",
				Properties: map[string]interface{}{
					"strength": 1.0,
				},
			},
		}

		err = client.InsertGraphEdges(ctx, edges)
		if err != nil {
			t.Logf("❌ graph_edges table not ready: %v", err)
			client.DeleteText(ctx, testText.ID) // 清理
			t.Fail()
			return
		}

		t.Log("✅ graph_edges table is working")
		require.NotEmpty(t, edges[0].ID)

		// 測試檢索
		foundEdges, err := client.GetEdgesByRelationType(ctx, "TEST_RELATION")
		require.NoError(t, err)
		assert.Len(t, foundEdges, 1)
		assert.Equal(t, "TEST_RELATION", foundEdges[0].RelationshipType)
		t.Log("✅ graph_edges table read operation working")

		// 清理
		client.DeleteText(ctx, testText.ID) // 這會級聯刪除相關數據
	})

	t.Run("VerifyEmbeddingsTable", func(t *testing.T) {
		// 先創建必要的數據
		testText := &models.TextRecord{
			Content: "Text for embeddings test",
			Title:   "Embeddings Test",
			Status:  "completed",
		}
		err := client.InsertText(ctx, testText)
		require.NoError(t, err)

		testChunk := &models.ChunkRecord{
			TextID:      testText.ID,
			Content:     "Chunk for embeddings",
			IndentLevel: 0,
		}
		err = client.InsertChunk(ctx, testChunk)
		require.NoError(t, err)

		// 創建 embedding
		embeddings := []models.EmbeddingRecord{
			{
				ChunkID: testChunk.ID,
				Vector:  []float64{0.1, 0.2, 0.3, 0.4, 0.5}, // 簡單的測試向量
			},
		}

		err = client.InsertEmbeddings(ctx, embeddings)
		if err != nil {
			t.Logf("❌ embeddings table not ready: %v", err)
			client.DeleteText(ctx, testText.ID) // 清理
			t.Fail()
			return
		}

		t.Log("✅ embeddings table is working")
		require.NotEmpty(t, embeddings[0].ID)

		// 清理
		client.DeleteText(ctx, testText.ID) // 這會級聯刪除相關數據
	})
}

func TestFullGraphWorkflow(t *testing.T) {
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

	t.Run("CompleteGraphWorkflow", func(t *testing.T) {
		// 1. 創建 text
		testText := &models.TextRecord{
			Content: "Complete workflow test content",
			Title:   "Complete Workflow Test",
			Status:  "completed",
		}
		err := client.InsertText(ctx, testText)
		require.NoError(t, err)
		t.Log("✅ Text created successfully")

		// 2. 創建 chunks
		chunks := []models.ChunkRecord{
			{
				TextID:      testText.ID,
				Content:     "Alice works at Microsoft",
				IndentLevel: 0,
			},
			{
				TextID:      testText.ID,
				Content:     "Microsoft is a technology company",
				IndentLevel: 0,
			},
		}
		err = client.InsertChunks(ctx, chunks)
		require.NoError(t, err)
		t.Log("✅ Chunks created successfully")

		// 3. 創建 graph nodes
		nodes := []models.GraphNode{
			{
				ChunkID:    chunks[0].ID,
				EntityName: "Alice",
				EntityType: "Person",
				Properties: map[string]interface{}{
					"role": "Employee",
				},
			},
			{
				ChunkID:    chunks[1].ID,
				EntityName: "Microsoft",
				EntityType: "Company",
				Properties: map[string]interface{}{
					"industry": "Technology",
				},
			},
		}
		err = client.InsertGraphNodes(ctx, nodes)
		require.NoError(t, err)
		t.Log("✅ Graph nodes created successfully")

		// 4. 創建 graph edges
		edges := []models.GraphEdge{
			{
				SourceNodeID:     nodes[0].ID,
				TargetNodeID:     nodes[1].ID,
				RelationshipType: "WORKS_FOR",
				Properties: map[string]interface{}{
					"since": "2020",
				},
			},
		}
		err = client.InsertGraphEdges(ctx, edges)
		require.NoError(t, err)
		t.Log("✅ Graph edges created successfully")

		// 5. 測試圖形搜尋
		result, err := client.SearchGraph(ctx, &models.GraphQuery{
			EntityName: "Alice",
			MaxDepth:   2,
			Limit:      10,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, result.Nodes)
		t.Logf("✅ Graph search successful, found %d nodes and %d edges", len(result.Nodes), len(result.Edges))

		// 6. 測試鄰居搜尋
		neighbors, err := client.GetNodeNeighbors(ctx, nodes[0].ID, 1)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(neighbors.Nodes), 1)
		t.Logf("✅ Neighbor search successful, found %d neighbors", len(neighbors.Nodes))

		// 7. 清理
		err = client.DeleteText(ctx, testText.ID)
		assert.NoError(t, err)
		t.Log("✅ Cleanup completed")
	})
}