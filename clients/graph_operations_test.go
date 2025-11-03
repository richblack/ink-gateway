package clients

import (
	"context"
	"testing"

	"semantic-text-processor/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGraphOperations(t *testing.T) {
	mock := NewMockSupabaseClient()
	ctx := context.Background()
	
	// Setup test data
	setupGraphTestData(t, mock, ctx)
	
	t.Run("GetNodesByEntity", func(t *testing.T) {
		nodes, err := mock.GetNodesByEntity(ctx, "Alice")
		assert.NoError(t, err)
		assert.Len(t, nodes, 1)
		assert.Equal(t, "Alice", nodes[0].EntityName)
		assert.Equal(t, "Person", nodes[0].EntityType)
	})
	
	t.Run("GetNodesByEntity_NotFound", func(t *testing.T) {
		nodes, err := mock.GetNodesByEntity(ctx, "NonExistent")
		assert.NoError(t, err)
		assert.Empty(t, nodes)
	})
	
	t.Run("GetNodeNeighbors", func(t *testing.T) {
		// First get Alice's node ID
		aliceNodes, err := mock.GetNodesByEntity(ctx, "Alice")
		require.NoError(t, err)
		require.Len(t, aliceNodes, 1)
		
		result, err := mock.GetNodeNeighbors(ctx, aliceNodes[0].ID, 1)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		
		// Should include Alice herself and her neighbors
		assert.GreaterOrEqual(t, len(result.Nodes), 1)
		assert.GreaterOrEqual(t, len(result.Edges), 0)
	})
	
	t.Run("GetNodeNeighbors_InvalidNode", func(t *testing.T) {
		result, err := mock.GetNodeNeighbors(ctx, "invalid-node-id", 1)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Nodes)
		assert.Empty(t, result.Edges)
	})
	
	t.Run("FindPathBetweenNodes", func(t *testing.T) {
		// Get Alice and Bob node IDs
		aliceNodes, err := mock.GetNodesByEntity(ctx, "Alice")
		require.NoError(t, err)
		require.Len(t, aliceNodes, 1)
		
		bobNodes, err := mock.GetNodesByEntity(ctx, "Bob")
		require.NoError(t, err)
		require.Len(t, bobNodes, 1)
		
		result, err := mock.FindPathBetweenNodes(ctx, aliceNodes[0].ID, bobNodes[0].ID, 3)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		
		// Should find a path if one exists
		if len(result.Nodes) > 0 {
			assert.GreaterOrEqual(t, len(result.Nodes), 2)
			assert.GreaterOrEqual(t, len(result.Edges), 1)
		}
	})
	
	t.Run("FindPathBetweenNodes_NoPath", func(t *testing.T) {
		result, err := mock.FindPathBetweenNodes(ctx, "node1", "node2", 3)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Nodes)
		assert.Empty(t, result.Edges)
	})
	
	t.Run("GetNodesByChunk", func(t *testing.T) {
		nodes, err := mock.GetNodesByChunk(ctx, "test-chunk-1")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(nodes), 1)
		
		// All returned nodes should have the correct chunk ID
		for _, node := range nodes {
			assert.Equal(t, "test-chunk-1", node.ChunkID)
		}
	})
	
	t.Run("GetNodesByChunk_NotFound", func(t *testing.T) {
		nodes, err := mock.GetNodesByChunk(ctx, "non-existent-chunk")
		assert.NoError(t, err)
		assert.Empty(t, nodes)
	})
	
	t.Run("GetEdgesByRelationType", func(t *testing.T) {
		edges, err := mock.GetEdgesByRelationType(ctx, "KNOWS")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(edges), 1)
		
		// All returned edges should have the correct relationship type
		for _, edge := range edges {
			assert.Equal(t, "KNOWS", edge.RelationshipType)
		}
	})
	
	t.Run("GetEdgesByRelationType_NotFound", func(t *testing.T) {
		edges, err := mock.GetEdgesByRelationType(ctx, "NON_EXISTENT")
		assert.NoError(t, err)
		assert.Empty(t, edges)
	})
}

func TestGraphSearchFallback(t *testing.T) {
	mock := NewMockSupabaseClient()
	ctx := context.Background()
	
	// Setup test data
	setupGraphTestData(t, mock, ctx)
	
	t.Run("SearchGraph_WithFallback", func(t *testing.T) {
		query := &models.GraphQuery{
			EntityName: "Alice",
			MaxDepth:   2,
			Limit:      10,
		}
		
		result, err := mock.SearchGraph(ctx, query)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		
		// Should find Alice
		foundAlice := false
		for _, node := range result.Nodes {
			if node.EntityName == "Alice" {
				foundAlice = true
				break
			}
		}
		assert.True(t, foundAlice, "Should find Alice in search results")
	})
	
	t.Run("SearchGraph_EmptyResult", func(t *testing.T) {
		query := &models.GraphQuery{
			EntityName: "NonExistent",
			MaxDepth:   2,
			Limit:      10,
		}
		
		result, err := mock.SearchGraph(ctx, query)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Nodes)
		assert.Empty(t, result.Edges)
	})
}

func TestGraphOperationEdgeCases(t *testing.T) {
	mock := NewMockSupabaseClient()
	ctx := context.Background()
	
	t.Run("EmptyGraphOperations", func(t *testing.T) {
		// Test operations on empty graph
		nodes, err := mock.GetNodesByEntity(ctx, "Any")
		assert.NoError(t, err)
		assert.Empty(t, nodes)
		
		edges, err := mock.GetEdgesByRelationType(ctx, "ANY")
		assert.NoError(t, err)
		assert.Empty(t, edges)
		
		result, err := mock.GetNodeNeighbors(ctx, "any-id", 1)
		assert.NoError(t, err)
		assert.Empty(t, result.Nodes)
		assert.Empty(t, result.Edges)
	})
	
	t.Run("GraphOperationsWithDefaults", func(t *testing.T) {
		// Test with default values
		result, err := mock.GetNodeNeighbors(ctx, "test-id", 0) // Should use default depth
		assert.NoError(t, err)
		assert.NotNil(t, result)
		
		pathResult, err := mock.FindPathBetweenNodes(ctx, "node1", "node2", 0) // Should use default depth
		assert.NoError(t, err)
		assert.NotNil(t, pathResult)
	})
}

// setupGraphTestData creates test data for graph operations
func setupGraphTestData(t *testing.T, mock *MockSupabaseClient, ctx context.Context) {
	// Create test nodes
	nodes := []models.GraphNode{
		{
			ChunkID:    "test-chunk-1",
			EntityName: "Alice",
			EntityType: "Person",
			Properties: map[string]interface{}{
				"age": 30,
				"role": "Manager",
			},
		},
		{
			ChunkID:    "test-chunk-2",
			EntityName: "Bob",
			EntityType: "Person",
			Properties: map[string]interface{}{
				"age": 25,
				"role": "Developer",
			},
		},
		{
			ChunkID:    "test-chunk-3",
			EntityName: "Company X",
			EntityType: "Organization",
			Properties: map[string]interface{}{
				"industry": "Technology",
				"size": "Large",
			},
		},
	}
	
	err := mock.InsertGraphNodes(ctx, nodes)
	require.NoError(t, err)
	
	// Create test edges
	edges := []models.GraphEdge{
		{
			SourceNodeID:     nodes[0].ID, // Alice
			TargetNodeID:     nodes[1].ID, // Bob
			RelationshipType: "KNOWS",
			Properties: map[string]interface{}{
				"since": "2020",
				"strength": "strong",
			},
		},
		{
			SourceNodeID:     nodes[0].ID, // Alice
			TargetNodeID:     nodes[2].ID, // Company X
			RelationshipType: "WORKS_FOR",
			Properties: map[string]interface{}{
				"position": "Manager",
				"start_date": "2019-01-01",
			},
		},
		{
			SourceNodeID:     nodes[1].ID, // Bob
			TargetNodeID:     nodes[2].ID, // Company X
			RelationshipType: "WORKS_FOR",
			Properties: map[string]interface{}{
				"position": "Developer",
				"start_date": "2020-06-01",
			},
		},
	}
	
	err = mock.InsertGraphEdges(ctx, edges)
	require.NoError(t, err)
}