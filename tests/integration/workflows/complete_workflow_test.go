package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"semantic-text-processor/models"
	"semantic-text-processor/tests/framework/runner"
	"semantic-text-processor/tests/framework/testdata"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// CompleteWorkflowTestSuite tests end-to-end workflows
type CompleteWorkflowTestSuite struct {
	suite.Suite
	ctx              context.Context
	baseURL          string
	httpClient       *http.Client
	testDataManager  *testdata.Manager
	cleanupFunctions []func() error
}

// SetupSuite initializes the test suite
func (suite *CompleteWorkflowTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.baseURL = "http://localhost:8080" // Default test server
	suite.httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	// Initialize test data manager
	config := &testdata.Config{
		DataDirectory:   "./fixtures",
		SnapshotDir:     "./snapshots",
		MaxDataSetSize:  1000,
		CleanupAfterUse: true,
	}
	suite.testDataManager = testdata.NewManager(config)
	suite.cleanupFunctions = make([]func() error, 0)
}

// TearDownSuite cleans up after all tests
func (suite *CompleteWorkflowTestSuite) TearDownSuite() {
	// Execute cleanup functions
	for _, cleanup := range suite.cleanupFunctions {
		if err := cleanup(); err != nil {
			suite.T().Logf("Cleanup error: %v", err)
		}
	}
}

// TestCompleteTextProcessingWorkflow tests the full text processing pipeline
func (suite *CompleteWorkflowTestSuite) TestCompleteTextProcessingWorkflow() {
	suite.T().Log("Starting complete text processing workflow test")

	// Test cases with different text sizes and types
	testCases := []struct {
		name           string
		textContent    string
		expectedChunks int
		expectedNodes  int
	}{
		{
			name:           "Small Article",
			textContent:    suite.generateTestArticle(500),
			expectedChunks: 2,
			expectedNodes:  5,
		},
		{
			name:           "Medium Document",
			textContent:    suite.generateTestDocument(1500),
			expectedChunks: 5,
			expectedNodes:  15,
		},
		{
			name:           "Large Technical Content",
			textContent:    suite.generateTechnicalContent(3000),
			expectedChunks: 10,
			expectedNodes:  25,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			suite.runTextProcessingWorkflow(tc.textContent, tc.expectedChunks, tc.expectedNodes)
		})
	}
}

// runTextProcessingWorkflow executes a complete text processing workflow
func (suite *CompleteWorkflowTestSuite) runTextProcessingWorkflow(content string, expectedChunks, expectedNodes int) {
	// Step 1: Submit text for processing
	textRecord := &models.TextRecord{
		Content:  content,
		Title:    "Test Document",
		Source:   "integration_test",
		Language: "en",
		Status:   "pending",
		Metadata: map[string]interface{}{
			"test_case": "complete_workflow",
		},
	}

	textID, err := suite.submitText(textRecord)
	suite.Require().NoError(err, "Failed to submit text")
	suite.Require().NotEmpty(textID, "Text ID should not be empty")

	// Register cleanup
	suite.cleanupFunctions = append(suite.cleanupFunctions, func() error {
		return suite.deleteText(textID)
	})

	// Step 2: Wait for processing to complete
	suite.waitForProcessingComplete(textID, 60*time.Second)

	// Step 3: Verify text record
	retrievedText, err := suite.getText(textID)
	suite.Require().NoError(err, "Failed to retrieve text")
	suite.Assert().Equal("processed", retrievedText.Status, "Text should be processed")
	suite.Assert().Equal(content, retrievedText.Content, "Content should match")

	// Step 4: Verify chunks were created
	chunks, err := suite.getChunksByTextID(textID)
	suite.Require().NoError(err, "Failed to retrieve chunks")
	suite.Assert().GreaterOrEqual(len(chunks), expectedChunks, "Should have expected number of chunks")

	// Verify chunk content integrity
	totalContent := ""
	for _, chunk := range chunks {
		suite.Assert().NotEmpty(chunk.Content, "Chunk content should not be empty")
		suite.Assert().Equal(textID, chunk.TextID, "Chunk should reference correct text")
		totalContent += chunk.Content
	}

	// Step 5: Verify embeddings were generated
	for _, chunk := range chunks {
		embedding, err := suite.getEmbeddingByChunkID(chunk.ID)
		suite.Require().NoError(err, "Failed to retrieve embedding for chunk %s", chunk.ID)
		suite.Assert().NotEmpty(embedding.Vector, "Embedding vector should not be empty")
		suite.Assert().Greater(len(embedding.Vector), 100, "Embedding should have reasonable dimensions")
	}

	// Step 6: Verify knowledge graph was created
	graphNodes, err := suite.getGraphNodesByTextID(textID)
	suite.Require().NoError(err, "Failed to retrieve graph nodes")
	suite.Assert().GreaterOrEqual(len(graphNodes), expectedNodes, "Should have expected number of graph nodes")

	// Verify graph structure
	for _, node := range graphNodes {
		suite.Assert().NotEmpty(node.EntityName, "Node should have entity name")
		suite.Assert().NotEmpty(node.EntityType, "Node should have entity type")
		suite.Assert().NotEmpty(node.ChunkID, "Node should reference a chunk")
	}

	// Verify graph edges exist
	if len(graphNodes) > 1 {
		graphEdges, err := suite.getGraphEdgesByNodes(graphNodes)
		suite.Require().NoError(err, "Failed to retrieve graph edges")
		suite.Assert().Greater(len(graphEdges), 0, "Should have graph edges connecting nodes")
	}

	// Step 7: Test semantic search functionality
	suite.testSemanticSearchWithProcessedContent(textID, content)
}

// TestSearchAndRetrievalWorkflow tests comprehensive search functionality
func (suite *CompleteWorkflowTestSuite) TestSearchAndRetrievalWorkflow() {
	suite.T().Log("Starting search and retrieval workflow test")

	// Prepare test data
	schema := testdata.GetDefaultSchemas()["medium"]
	dataset, err := suite.testDataManager.GenerateData(schema)
	suite.Require().NoError(err, "Failed to generate test data")

	// Seed database with test data
	err = suite.testDataManager.SeedDatabase(suite.ctx, dataset)
	suite.Require().NoError(err, "Failed to seed database")

	// Register cleanup
	suite.cleanupFunctions = append(suite.cleanupFunctions, func() error {
		return suite.testDataManager.CleanData("generated_*")
	})

	// Test different search scenarios
	searchTests := []struct {
		name           string
		query          string
		searchType     string
		expectedMin    int
		validateResult func(results []models.SimilarityResult)
	}{
		{
			name:        "Keyword Search",
			query:       "test sample data",
			searchType:  "semantic",
			expectedMin: 1,
			validateResult: func(results []models.SimilarityResult) {
				// Verify results contain relevant content
				for _, result := range results {
					content := strings.ToLower(result.Chunk.Content)
					suite.Assert().True(
						strings.Contains(content, "test") ||
							strings.Contains(content, "sample") ||
							strings.Contains(content, "data"),
						"Result should contain query keywords",
					)
				}
			},
		},
		{
			name:        "Conceptual Search",
			query:       "system architecture methodology",
			searchType:  "semantic",
			expectedMin: 2,
			validateResult: func(results []models.SimilarityResult) {
				// Verify semantic relevance
				for _, result := range results {
					suite.Assert().Greater(result.Similarity, 0.1, "Should have meaningful similarity score")
				}
			},
		},
		{
			name:        "Hybrid Search",
			query:       "implementation process",
			searchType:  "hybrid",
			expectedMin: 3,
			validateResult: func(results []models.SimilarityResult) {
				// Verify ranking order
				for i := 1; i < len(results); i++ {
					suite.Assert().GreaterOrEqual(
						results[i-1].Similarity,
						results[i].Similarity,
						"Results should be ordered by similarity",
					)
				}
			},
		},
	}

	for _, test := range searchTests {
		suite.T().Run(test.name, func(t *testing.T) {
			// Execute search
			results, err := suite.executeSearch(test.query, test.searchType, 20)
			suite.Require().NoError(err, "Search should execute successfully")
			suite.Assert().GreaterOrEqual(len(results), test.expectedMin, "Should return minimum expected results")

			// Validate results
			if test.validateResult != nil {
				test.validateResult(results)
			}

			// Performance validation
			startTime := time.Now()
			_, err = suite.executeSearch(test.query, test.searchType, 10)
			duration := time.Since(startTime)
			suite.Assert().Less(duration, 500*time.Millisecond, "Search should complete within performance threshold")
		})
	}
}

// TestTemplateManagementWorkflow tests template creation and management
func (suite *CompleteWorkflowTestSuite) TestTemplateManagementWorkflow() {
	suite.T().Log("Starting template management workflow test")

	// Test template creation
	templateReq := &models.CreateTemplateRequest{
		Name:      "Test Report Template",
		Content:   "Report: {{title}}\nAuthor: {{author}}\nDate: {{date}}\nSummary: {{summary}}",
		SlotNames: []string{"title", "author", "date", "summary"},
		Metadata: map[string]interface{}{
			"category": "report",
			"version":  "1.0",
		},
	}

	template, err := suite.createTemplate(templateReq)
	suite.Require().NoError(err, "Failed to create template")
	suite.Assert().NotEmpty(template.ID, "Template should have ID")

	// Register cleanup
	suite.cleanupFunctions = append(suite.cleanupFunctions, func() error {
		return suite.deleteTemplate(template.ID)
	})

	// Test template instance creation
	instanceReq := &models.CreateInstanceRequest{
		TemplateID: template.ID,
		SlotValues: map[string]string{
			"title":   "Quarterly Report",
			"author":  "Test Author",
			"date":    "2024-01-01",
			"summary": "This is a test summary of the quarterly report.",
		},
		Metadata: map[string]interface{}{
			"quarter": "Q1 2024",
		},
	}

	instance, err := suite.createTemplateInstance(instanceReq)
	suite.Require().NoError(err, "Failed to create template instance")
	suite.Assert().NotEmpty(instance.ID, "Instance should have ID")

	// Verify instance content
	suite.Assert().Contains(instance.RenderedContent, "Quarterly Report", "Instance should contain title")
	suite.Assert().Contains(instance.RenderedContent, "Test Author", "Instance should contain author")

	// Test slot value updates
	err = suite.updateSlotValue(instance.ID, "summary", "Updated summary content")
	suite.Require().NoError(err, "Failed to update slot value")

	// Verify update
	updatedInstance, err := suite.getTemplateInstance(instance.ID)
	suite.Require().NoError(err, "Failed to retrieve updated instance")
	suite.Assert().Contains(updatedInstance.RenderedContent, "Updated summary content", "Instance should reflect updates")

	// Test template search and retrieval
	templates, err := suite.getAllTemplates()
	suite.Require().NoError(err, "Failed to retrieve templates")
	suite.Assert().Greater(len(templates), 0, "Should have templates")

	// Find our created template
	found := false
	for _, t := range templates {
		if t.ID == template.ID {
			found = true
			suite.Assert().Equal(templateReq.Name, t.Name, "Template name should match")
			break
		}
	}
	suite.Assert().True(found, "Created template should be found in list")
}

// TestErrorRecoveryWorkflow tests system behavior under error conditions
func (suite *CompleteWorkflowTestSuite) TestErrorRecoveryWorkflow() {
	suite.T().Log("Starting error recovery workflow test")

	// Test cases for different error scenarios
	errorTests := []struct {
		name          string
		errorType     string
		setupError    func()
		cleanupError  func()
		testOperation func() error
		expectRecovery bool
	}{
		{
			name:      "Database Connection Error",
			errorType: "database",
			setupError: func() {
				// Simulate database connection issues
				suite.simulateDatabaseError()
			},
			cleanupError: func() {
				suite.restoreDatabaseConnection()
			},
			testOperation: func() error {
				_, err := suite.submitText(&models.TextRecord{
					Content: "Test content for error recovery",
					Title:   "Error Test",
				})
				return err
			},
			expectRecovery: true,
		},
		{
			name:      "LLM Service Timeout",
			errorType: "external_service",
			setupError: func() {
				suite.simulateLLMTimeout()
			},
			cleanupError: func() {
				suite.restoreLLMService()
			},
			testOperation: func() error {
				return suite.processTextWithLLM("Test content for LLM timeout")
			},
			expectRecovery: true,
		},
	}

	for _, test := range errorTests {
		suite.T().Run(test.name, func(t *testing.T) {
			// Setup error condition
			test.setupError()

			// Test that operation fails initially
			err := test.testOperation()
			suite.Assert().Error(err, "Operation should fail under error condition")

			// Restore service
			test.cleanupError()

			if test.expectRecovery {
				// Wait for recovery
				time.Sleep(2 * time.Second)

				// Test that operation succeeds after recovery
				err = test.testOperation()
				suite.Assert().NoError(err, "Operation should succeed after recovery")
			}
		})
	}
}

// Helper methods for API interactions

func (suite *CompleteWorkflowTestSuite) submitText(text *models.TextRecord) (string, error) {
	jsonData, err := json.Marshal(text)
	if err != nil {
		return "", err
	}

	resp, err := suite.httpClient.Post(
		suite.baseURL+"/api/texts",
		"application/json",
		strings.NewReader(string(jsonData)),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.ID, nil
}

func (suite *CompleteWorkflowTestSuite) getText(textID string) (*models.TextDetail, error) {
	resp, err := suite.httpClient.Get(suite.baseURL + "/api/texts/" + textID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var text models.TextDetail
	if err := json.NewDecoder(resp.Body).Decode(&text); err != nil {
		return nil, err
	}

	return &text, nil
}

func (suite *CompleteWorkflowTestSuite) getChunksByTextID(textID string) ([]models.ChunkRecord, error) {
	resp, err := suite.httpClient.Get(suite.baseURL + "/api/texts/" + textID + "/chunks")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var chunks []models.ChunkRecord
	if err := json.NewDecoder(resp.Body).Decode(&chunks); err != nil {
		return nil, err
	}

	return chunks, nil
}

func (suite *CompleteWorkflowTestSuite) getEmbeddingByChunkID(chunkID string) (*models.EmbeddingRecord, error) {
	resp, err := suite.httpClient.Get(suite.baseURL + "/api/chunks/" + chunkID + "/embedding")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var embedding models.EmbeddingRecord
	if err := json.NewDecoder(resp.Body).Decode(&embedding); err != nil {
		return nil, err
	}

	return &embedding, nil
}

func (suite *CompleteWorkflowTestSuite) executeSearch(query, searchType string, limit int) ([]models.SimilarityResult, error) {
	searchReq := map[string]interface{}{
		"query": query,
		"limit": limit,
		"type":  searchType,
	}

	jsonData, err := json.Marshal(searchReq)
	if err != nil {
		return nil, err
	}

	resp, err := suite.httpClient.Post(
		suite.baseURL+"/api/search/semantic",
		"application/json",
		strings.NewReader(string(jsonData)),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var results []models.SimilarityResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	return results, nil
}

// Helper methods for test data generation

func (suite *CompleteWorkflowTestSuite) generateTestArticle(length int) string {
	content := "Introduction\n\n"
	content += "This is a test article for integration testing. "
	content += strings.Repeat("This article contains various concepts and ideas that will be processed by the semantic text processor. ", length/100)
	content += "\n\nConclusion\n\n"
	content += "This article demonstrates the complete text processing workflow including chunking, embedding generation, and knowledge graph extraction."
	return content
}

func (suite *CompleteWorkflowTestSuite) generateTestDocument(length int) string {
	content := "Technical Documentation\n\n"
	content += "Overview: This document describes the system architecture and implementation details. "
	content += strings.Repeat("The system consists of multiple components including text processing, embedding generation, search capabilities, and knowledge graph construction. ", length/150)
	content += "\n\nImplementation Details\n\n"
	content += "The implementation uses modern software engineering practices and follows established design patterns."
	return content
}

func (suite *CompleteWorkflowTestSuite) generateTechnicalContent(length int) string {
	content := "System Specification\n\n"
	content += "Architecture: The semantic text processor implements a microservices architecture with dedicated services for text processing, embedding generation, and search. "
	content += strings.Repeat("Each component is designed for scalability and maintainability, with clear interfaces and well-defined responsibilities. ", length/200)
	content += "\n\nPerformance Characteristics\n\n"
	content += "The system is designed to handle high throughput while maintaining low latency for user queries."
	return content
}

// Error simulation methods

func (suite *CompleteWorkflowTestSuite) simulateDatabaseError() {
	// Implementation would configure mock to simulate database errors
}

func (suite *CompleteWorkflowTestSuite) restoreDatabaseConnection() {
	// Implementation would restore normal database connectivity
}

func (suite *CompleteWorkflowTestSuite) simulateLLMTimeout() {
	// Implementation would configure mock to simulate LLM service timeouts
}

func (suite *CompleteWorkflowTestSuite) restoreLLMService() {
	// Implementation would restore normal LLM service operation
}

// Additional helper methods would be implemented here...

// waitForProcessingComplete waits for text processing to complete
func (suite *CompleteWorkflowTestSuite) waitForProcessingComplete(textID string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		text, err := suite.getText(textID)
		if err == nil && text.Status == "processed" {
			return
		}
		time.Sleep(1 * time.Second)
	}

	suite.Fail("Text processing did not complete within timeout")
}

// testSemanticSearchWithProcessedContent tests search functionality with processed content
func (suite *CompleteWorkflowTestSuite) testSemanticSearchWithProcessedContent(textID, content string) {
	// Extract key phrases from content for search testing
	words := strings.Fields(content)
	if len(words) > 10 {
		query := strings.Join(words[5:10], " ") // Use middle portion as query

		results, err := suite.executeSearch(query, "semantic", 10)
		suite.Require().NoError(err, "Search should succeed")

		// Verify that our processed text appears in results
		found := false
		for _, result := range results {
			if strings.Contains(result.Chunk.Content, query) {
				found = true
				break
			}
		}
		suite.Assert().True(found, "Processed content should be findable via search")
	}
}

// Run the test suite
func TestCompleteWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(CompleteWorkflowTestSuite))
}