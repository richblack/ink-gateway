package performance

import (
	"semantic-text-processor/models"
	"time"
)

// ComprehensivePerformanceReport represents the complete performance test results
type ComprehensivePerformanceReport struct {
	StartTime              time.Time                           `json:"start_time"`
	EndTime                time.Time                           `json:"end_time"`
	TotalDuration          time.Duration                       `json:"total_duration"`
	TestConfig             PerformanceTestConfig               `json:"test_config"`
	Environment            models.EnvironmentInfo              `json:"environment"`
	DataGeneration         models.DataGenerationResult         `json:"data_generation"`
	BaselinePerformance    models.BaselinePerformanceResult    `json:"baseline_performance"`
	LoadTestResults        models.LoadTestResult               `json:"load_test_results"`
	OptimizationAnalysis   models.OptimizationAnalysisResult   `json:"optimization_analysis"`
	RegressionResults      *models.RegressionTestResult        `json:"regression_results,omitempty"`
	ResourceUtilization    models.ResourceUtilizationResult    `json:"resource_utilization"`
	Recommendations        []models.PerformanceRecommendation  `json:"recommendations"`
}

// TestQuery represents a test query for performance testing
type TestQuery struct {
	Type       string      `json:"type"`
	Parameters interface{} `json:"parameters"`
	Expected   interface{} `json:"expected,omitempty"`
}

// Query generators
type SemanticSearchGenerator struct{}
type TagSearchGenerator struct{}
type ChunkCRUDGenerator struct{}

func NewSemanticSearchGenerator() *SemanticSearchGenerator {
	return &SemanticSearchGenerator{}
}

func (g *SemanticSearchGenerator) GenerateQuery() models.TestQuery {
	queries := []string{
		"machine learning algorithms",
		"database optimization techniques",
		"performance monitoring tools",
		"web application security",
		"cloud computing services",
		"data analysis methods",
		"software architecture patterns",
		"API design best practices",
	}

	query := queries[int(time.Now().UnixNano())%len(queries)]

	return models.TestQuery{
		Type: "semantic_search",
		Parameters: &models.OptimizedSearchRequest{
			Query:         query,
			Limit:         10,
			MinSimilarity: 0.7,
			UseCache:      true,
		},
	}
}

func (g *SemanticSearchGenerator) GetQueryType() string {
	return "semantic_search"
}

func NewTagSearchGenerator() *TagSearchGenerator {
	return &TagSearchGenerator{}
}

func (g *TagSearchGenerator) GenerateQuery() models.TestQuery {
	tagSets := [][]string{
		{"programming", "python"},
		{"database", "optimization"},
		{"web", "security"},
		{"cloud", "aws"},
		{"machine-learning", "ai"},
		{"performance", "monitoring"},
		{"architecture", "microservices"},
		{"testing", "automation"},
	}

	tags := tagSets[int(time.Now().UnixNano())%len(tagSets)]

	return models.TestQuery{
		Type: "tag_search",
		Parameters: &models.TagSearchRequest{
			Tags:            tags,
			CombinationMode: "AND",
			Limit:          20,
		},
	}
}

func (g *TagSearchGenerator) GetQueryType() string {
	return "tag_search"
}

func NewChunkCRUDGenerator() *ChunkCRUDGenerator {
	return &ChunkCRUDGenerator{}
}

func (g *ChunkCRUDGenerator) GenerateQuery() models.TestQuery {
	return models.TestQuery{
		Type: "chunk_crud",
		Parameters: map[string]interface{}{
			"operation": "read",
			"limit":     20,
			"offset":    0,
		},
	}
}

func (g *ChunkCRUDGenerator) GetQueryType() string {
	return "chunk_crud"
}