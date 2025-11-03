package clients

import (
	"context"
	"fmt"
	"testing"

	"semantic-text-processor/models"

	"github.com/stretchr/testify/assert"
)

// Note: MockSupabaseClient is defined in supabase_test.go to avoid duplication

// Graph operations for MockSupabaseClient

func (m *MockSupabaseClient) InsertGraphNodes(ctx context.Context, nodes []models.GraphNode) error {
	for i := range nodes {
		if nodes[i].ID == "" {
			nodes[i].ID = generateUUID()
		}
		if nodes[i].Properties == nil {
			nodes[i].Properties = make(map[string]interface{})
		}
	}
	m.nodes = append(m.nodes, nodes...)
	return nil
}

func (m *MockSupabaseClient) InsertGraphEdges(ctx context.Context, edges []models.GraphEdge) error {
	for i := range edges {
		if edges[i].ID == "" {
			edges[i].ID = generateUUID()
		}
		if edges[i].Properties == nil {
			edges[i].Properties = make(map[string]interface{})
		}
	}
	m.edges = append(m.edges, edges...)
	return nil
}

func (m *MockSupabaseClient) SearchGraph(ctx context.Context, query *models.GraphQuery) (*models.GraphResult, error) {
	if query == nil {
		return nil, fmt.Errorf("graph query cannot be nil")
	}
	
	// Simple mock search - find nodes by entity name
	var resultNodes []models.GraphNode
	var resultEdges []models.GraphEdge
	
	for _, node := range m.nodes {
		if node.EntityName == query.EntityName {
			resultNodes = append(resultNodes, node)
		}
	}
	
	// Find edges connected to found nodes
	for _, node := range resultNodes {
		for _, edge := range m.edges {
			if edge.SourceNodeID == node.ID || edge.TargetNodeID == node.ID {
				resultEdges = append(resultEdges, edge)
			}
		}
	}
	
	return &models.GraphResult{
		Nodes: resultNodes,
		Edges: resultEdges,
	}, nil
}

// Additional graph operations for MockSupabaseClient

func (m *MockSupabaseClient) GetNodesByEntity(ctx context.Context, entityName string) ([]models.GraphNode, error) {
	var result []models.GraphNode
	for _, node := range m.nodes {
		if node.EntityName == entityName {
			result = append(result, node)
		}
	}
	return result, nil
}

func (m *MockSupabaseClient) GetNodeNeighbors(ctx context.Context, nodeID string, maxDepth int) (*models.GraphResult, error) {
	// Simple implementation for testing
	var resultNodes []models.GraphNode
	var resultEdges []models.GraphEdge
	
	// Find the starting node
	var startNode *models.GraphNode
	for _, node := range m.nodes {
		if node.ID == nodeID {
			startNode = &node
			break
		}
	}
	
	if startNode == nil {
		return &models.GraphResult{Nodes: []models.GraphNode{}, Edges: []models.GraphEdge{}}, nil
	}
	
	resultNodes = append(resultNodes, *startNode)
	
	// Find direct neighbors (depth 1 for simplicity)
	for _, edge := range m.edges {
		var neighborID string
		if edge.SourceNodeID == nodeID {
			neighborID = edge.TargetNodeID
		} else if edge.TargetNodeID == nodeID {
			neighborID = edge.SourceNodeID
		}
		
		if neighborID != "" {
			resultEdges = append(resultEdges, edge)
			for _, node := range m.nodes {
				if node.ID == neighborID {
					resultNodes = append(resultNodes, node)
					break
				}
			}
		}
	}
	
	return &models.GraphResult{Nodes: resultNodes, Edges: resultEdges}, nil
}

func (m *MockSupabaseClient) FindPathBetweenNodes(ctx context.Context, sourceNodeID, targetNodeID string, maxDepth int) (*models.GraphResult, error) {
	// Simple implementation - just check if there's a direct edge
	for _, edge := range m.edges {
		if (edge.SourceNodeID == sourceNodeID && edge.TargetNodeID == targetNodeID) ||
			(edge.SourceNodeID == targetNodeID && edge.TargetNodeID == sourceNodeID) {
			
			var nodes []models.GraphNode
			for _, node := range m.nodes {
				if node.ID == sourceNodeID || node.ID == targetNodeID {
					nodes = append(nodes, node)
				}
			}
			
			return &models.GraphResult{
				Nodes: nodes,
				Edges: []models.GraphEdge{edge},
			}, nil
		}
	}
	
	return &models.GraphResult{Nodes: []models.GraphNode{}, Edges: []models.GraphEdge{}}, nil
}

func (m *MockSupabaseClient) GetNodesByChunk(ctx context.Context, chunkID string) ([]models.GraphNode, error) {
	var result []models.GraphNode
	for _, node := range m.nodes {
		if node.ChunkID == chunkID {
			result = append(result, node)
		}
	}
	return result, nil
}

func (m *MockSupabaseClient) GetEdgesByRelationType(ctx context.Context, relationType string) ([]models.GraphEdge, error) {
	var result []models.GraphEdge
	for _, edge := range m.edges {
		if edge.RelationshipType == relationType {
			result = append(result, edge)
		}
	}
	return result, nil
}

func TestMockGraphOperations(t *testing.T) {
	mock := NewMockSupabaseClient()
	ctx := context.Background()
	
	t.Run("MockInsertGraphNodes", func(t *testing.T) {
		nodes := []models.GraphNode{
			{
				ChunkID:    "test-chunk-1",
				EntityName: "Alice",
				EntityType: "Person",
				Properties: map[string]interface{}{
					"age": 30,
				},
			},
			{
				ChunkID:    "test-chunk-2",
				EntityName: "Bob",
				EntityType: "Person",
				Properties: map[string]interface{}{
					"age": 25,
				},
			},
		}
		
		err := mock.InsertGraphNodes(ctx, nodes)
		assert.NoError(t, err)
		
		// Verify nodes were added with IDs
		assert.Len(t, mock.nodes, 2)
		for _, node := range nodes {
			assert.NotEmpty(t, node.ID)
		}
	})
	
	t.Run("MockInsertGraphEdges", func(t *testing.T) {
		// First add nodes
		nodes := []models.GraphNode{
			{
				ChunkID:    "test-chunk-1",
				EntityName: "Charlie",
				EntityType: "Person",
			},
			{
				ChunkID:    "test-chunk-2",
				EntityName: "Diana",
				EntityType: "Person",
			},
		}
		
		err := mock.InsertGraphNodes(ctx, nodes)
		assert.NoError(t, err)
		
		// Add edge between them
		edges := []models.GraphEdge{
			{
				SourceNodeID:     nodes[0].ID,
				TargetNodeID:     nodes[1].ID,
				RelationshipType: "KNOWS",
				Properties: map[string]interface{}{
					"since": "2020",
				},
			},
		}
		
		err = mock.InsertGraphEdges(ctx, edges)
		assert.NoError(t, err)
		
		// Verify edge was added with ID
		assert.Len(t, mock.edges, 1)
		assert.NotEmpty(t, edges[0].ID)
	})
	
	t.Run("MockSearchGraph", func(t *testing.T) {
		// Add test data
		nodes := []models.GraphNode{
			{
				ChunkID:    "test-chunk-1",
				EntityName: "Eve",
				EntityType: "Person",
			},
		}
		
		err := mock.InsertGraphNodes(ctx, nodes)
		assert.NoError(t, err)
		
		// Search for the node
		query := &models.GraphQuery{
			EntityName: "Eve",
			MaxDepth:   2,
			Limit:      10,
		}
		
		result, err := mock.SearchGraph(ctx, query)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Nodes, 1)
		assert.Equal(t, "Eve", result.Nodes[0].EntityName)
	})
	
	t.Run("MockSearchGraph_WithNilQuery", func(t *testing.T) {
		result, err := mock.SearchGraph(ctx, nil)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "graph query cannot be nil")
	})
	
	t.Run("MockSearchGraph_NoResults", func(t *testing.T) {
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

func TestGraphOperationValidation(t *testing.T) {
	t.Run("ValidateGraphNodeFields", func(t *testing.T) {
		node := models.GraphNode{
			ChunkID:    "chunk-123",
			EntityName: "Test Entity",
			EntityType: "TestType",
			Properties: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
				"key3": true,
			},
		}
		
		// Validate required fields
		assert.NotEmpty(t, node.ChunkID)
		assert.NotEmpty(t, node.EntityName)
		assert.NotEmpty(t, node.EntityType)
		assert.NotNil(t, node.Properties)
		
		// Validate properties can hold different types
		assert.Equal(t, "value1", node.Properties["key1"])
		assert.Equal(t, 42, node.Properties["key2"])
		assert.Equal(t, true, node.Properties["key3"])
	})
	
	t.Run("ValidateGraphEdgeFields", func(t *testing.T) {
		edge := models.GraphEdge{
			SourceNodeID:     "source-123",
			TargetNodeID:     "target-456",
			RelationshipType: "RELATED_TO",
			Properties: map[string]interface{}{
				"weight":    0.8,
				"direction": "bidirectional",
			},
		}
		
		// Validate required fields
		assert.NotEmpty(t, edge.SourceNodeID)
		assert.NotEmpty(t, edge.TargetNodeID)
		assert.NotEmpty(t, edge.RelationshipType)
		assert.NotNil(t, edge.Properties)
		
		// Validate properties
		assert.Equal(t, 0.8, edge.Properties["weight"])
		assert.Equal(t, "bidirectional", edge.Properties["direction"])
	})
	
	t.Run("ValidateGraphQueryFields", func(t *testing.T) {
		query := models.GraphQuery{
			EntityName: "Test Entity",
			MaxDepth:   5,
			Limit:      100,
		}
		
		assert.NotEmpty(t, query.EntityName)
		assert.Greater(t, query.MaxDepth, 0)
		assert.Greater(t, query.Limit, 0)
	})
	
	t.Run("ValidateGraphResultStructure", func(t *testing.T) {
		result := models.GraphResult{
			Nodes: []models.GraphNode{
				{EntityName: "Node1", EntityType: "Type1"},
				{EntityName: "Node2", EntityType: "Type2"},
			},
			Edges: []models.GraphEdge{
				{RelationshipType: "CONNECTS"},
				{RelationshipType: "RELATES"},
			},
		}
		
		assert.Len(t, result.Nodes, 2)
		assert.Len(t, result.Edges, 2)
		
		// Validate node structure
		for _, node := range result.Nodes {
			assert.NotEmpty(t, node.EntityName)
			assert.NotEmpty(t, node.EntityType)
		}
		
		// Validate edge structure
		for _, edge := range result.Edges {
			assert.NotEmpty(t, edge.RelationshipType)
		}
	})
}