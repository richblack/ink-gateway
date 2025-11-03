package runner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"semantic-text-processor/tests/framework/metrics"
)

// TestRunner defines the interface for test execution
type TestRunner interface {
	Execute(suite TestSuite) (*TestResults, error)
	Parallel(suites []TestSuite) (*AggregateResults, error)
	Schedule(suite TestSuite, schedule CronSchedule) error
	Abort(testID string) error
}

// Engine implements TestRunner interface
type Engine struct {
	config          *Config
	metricsCollector *metrics.Collector
	activeTests     map[string]*TestExecution
	mutex           sync.RWMutex
}

// Config holds test runner configuration
type Config struct {
	MaxParallelTests int
	DefaultTimeout   time.Duration
	RetryAttempts    int
	MetricsEnabled   bool
}

// TestSuite defines a collection of related tests
type TestSuite interface {
	Name() string
	Setup() error
	TearDown() error
	Tests() []TestCase
	Config() *SuiteConfig
}

// TestCase defines an individual test
type TestCase interface {
	Name() string
	Execute(ctx TestContext) error
	Timeout() time.Duration
	Dependencies() []string
	Tags() []string
}

// TestExecution tracks execution state
type TestExecution struct {
	ID        string
	Suite     TestSuite
	StartTime time.Time
	Status    ExecutionStatus
	Results   *TestResults
	Context   context.Context
	Cancel    context.CancelFunc
}

// ExecutionStatus represents test execution state
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusPassed    ExecutionStatus = "passed"
	StatusFailed    ExecutionStatus = "failed"
	StatusAborted   ExecutionStatus = "aborted"
	StatusTimedOut  ExecutionStatus = "timed_out"
)

// TestResults contains test execution results
type TestResults struct {
	SuiteID      string
	TotalTests   int
	PassedTests  int
	FailedTests  int
	SkippedTests int
	Duration     time.Duration
	TestCases    []TestCaseResult
	Metrics      *metrics.TestMetrics
	Errors       []error
}

// TestCaseResult contains individual test case results
type TestCaseResult struct {
	Name       string
	Status     ExecutionStatus
	Duration   time.Duration
	Error      error
	Assertions []AssertionResult
	Metrics    map[string]interface{}
}

// AssertionResult contains assertion details
type AssertionResult struct {
	Description string
	Passed      bool
	Expected    interface{}
	Actual      interface{}
	Error       error
}

// NewEngine creates a new test runner engine
func NewEngine(config *Config) *TestRunner {
	engine := &Engine{
		config:          config,
		metricsCollector: metrics.NewCollector(),
		activeTests:     make(map[string]*TestExecution),
	}

	return engine
}

// Execute runs a single test suite
func (e *Engine) Execute(suite TestSuite) (*TestResults, error) {
	testID := generateTestID(suite.Name())

	// Create test execution context
	ctx, cancel := context.WithTimeout(context.Background(), e.getTimeout(suite))
	execution := &TestExecution{
		ID:        testID,
		Suite:     suite,
		StartTime: time.Now(),
		Status:    StatusRunning,
		Context:   ctx,
		Cancel:    cancel,
	}

	// Track active test
	e.mutex.Lock()
	e.activeTests[testID] = execution
	e.mutex.Unlock()

	defer func() {
		e.mutex.Lock()
		delete(e.activeTests, testID)
		e.mutex.Unlock()
		cancel()
	}()

	// Start metrics collection
	if e.config.MetricsEnabled {
		e.metricsCollector.StartCollection(testID)
		defer e.metricsCollector.StopCollection(testID)
	}

	// Execute the test suite
	results, err := e.executeSuite(ctx, suite)
	if err != nil {
		execution.Status = StatusFailed
		return nil, fmt.Errorf("test suite execution failed: %w", err)
	}

	execution.Status = StatusPassed
	execution.Results = results

	// Collect final metrics
	if e.config.MetricsEnabled {
		results.Metrics = e.metricsCollector.GetMetrics(testID)
	}

	return results, nil
}

// Parallel executes multiple test suites in parallel
func (e *Engine) Parallel(suites []TestSuite) (*AggregateResults, error) {
	if len(suites) == 0 {
		return &AggregateResults{}, nil
	}

	// Limit parallel execution
	maxParallel := e.config.MaxParallelTests
	if maxParallel <= 0 || maxParallel > len(suites) {
		maxParallel = len(suites)
	}

	results := make(chan *TestResults, len(suites))
	errors := make(chan error, len(suites))
	semaphore := make(chan struct{}, maxParallel)

	var wg sync.WaitGroup

	// Execute suites in parallel with concurrency limit
	for _, suite := range suites {
		wg.Add(1)
		go func(s TestSuite) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Execute test suite
			result, err := e.Execute(s)
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(suite)
	}

	// Wait for all tests to complete
	wg.Wait()
	close(results)
	close(errors)

	// Aggregate results
	aggregateResults := &AggregateResults{
		TotalSuites: len(suites),
		StartTime:   time.Now(),
	}

	for result := range results {
		aggregateResults.SuiteResults = append(aggregateResults.SuiteResults, result)
		aggregateResults.PassedSuites++
	}

	for err := range errors {
		aggregateResults.Errors = append(aggregateResults.Errors, err)
		aggregateResults.FailedSuites++
	}

	aggregateResults.EndTime = time.Now()
	aggregateResults.Duration = aggregateResults.EndTime.Sub(aggregateResults.StartTime)

	return aggregateResults, nil
}

// Schedule schedules a test suite for repeated execution
func (e *Engine) Schedule(suite TestSuite, schedule CronSchedule) error {
	// Implementation for scheduled test execution
	// This would integrate with a cron scheduler
	return fmt.Errorf("scheduled execution not implemented yet")
}

// Abort cancels a running test
func (e *Engine) Abort(testID string) error {
	e.mutex.RLock()
	execution, exists := e.activeTests[testID]
	e.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("test with ID %s not found", testID)
	}

	execution.Cancel()
	execution.Status = StatusAborted

	return nil
}

// executeSuite runs all tests in a suite
func (e *Engine) executeSuite(ctx context.Context, suite TestSuite) (*TestResults, error) {
	startTime := time.Now()

	// Setup suite
	if err := suite.Setup(); err != nil {
		return nil, fmt.Errorf("suite setup failed: %w", err)
	}

	defer func() {
		if err := suite.TearDown(); err != nil {
			fmt.Printf("Warning: suite teardown failed: %v\n", err)
		}
	}()

	// Get test cases
	testCases := suite.Tests()
	results := &TestResults{
		SuiteID:    suite.Name(),
		TotalTests: len(testCases),
		TestCases:  make([]TestCaseResult, 0, len(testCases)),
	}

	// Execute each test case
	for _, testCase := range testCases {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
			result := e.executeTestCase(ctx, testCase)
			results.TestCases = append(results.TestCases, result)

			switch result.Status {
			case StatusPassed:
				results.PassedTests++
			case StatusFailed:
				results.FailedTests++
			default:
				results.SkippedTests++
			}
		}
	}

	results.Duration = time.Since(startTime)
	return results, nil
}

// executeTestCase runs a single test case
func (e *Engine) executeTestCase(ctx context.Context, testCase TestCase) TestCaseResult {
	startTime := time.Now()

	// Create test context with timeout
	timeout := testCase.Timeout()
	if timeout == 0 {
		timeout = e.config.DefaultTimeout
	}

	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result := TestCaseResult{
		Name:    testCase.Name(),
		Metrics: make(map[string]interface{}),
	}

	// Execute test case
	err := testCase.Execute(NewTestContext(testCtx))
	result.Duration = time.Since(startTime)

	if err != nil {
		result.Status = StatusFailed
		result.Error = err
	} else {
		result.Status = StatusPassed
	}

	return result
}

// getTimeout determines timeout for a test suite
func (e *Engine) getTimeout(suite TestSuite) time.Duration {
	config := suite.Config()
	if config != nil && config.Timeout > 0 {
		return config.Timeout
	}
	return e.config.DefaultTimeout
}

// generateTestID creates a unique identifier for test execution
func generateTestID(suiteName string) string {
	return fmt.Sprintf("%s_%d", suiteName, time.Now().Unix())
}

// AggregateResults contains results from multiple test suites
type AggregateResults struct {
	TotalSuites   int
	PassedSuites  int
	FailedSuites  int
	SuiteResults  []*TestResults
	Errors        []error
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
}

// SuiteConfig contains suite-specific configuration
type SuiteConfig struct {
	Timeout     time.Duration
	Parallel    bool
	Tags        []string
	Environment string
}

// CronSchedule defines scheduled execution
type CronSchedule struct {
	Expression string
	Timezone   string
}

// TestContext provides context for test execution
type TestContext interface {
	Context() context.Context
	Assert() *Asserter
	Log(message string)
	SetMetric(key string, value interface{})
	GetMetric(key string) interface{}
}

// DefaultTestContext implements TestContext
type DefaultTestContext struct {
	ctx     context.Context
	asserter *Asserter
	metrics map[string]interface{}
	logs    []string
}

// NewTestContext creates a new test context
func NewTestContext(ctx context.Context) TestContext {
	return &DefaultTestContext{
		ctx:     ctx,
		asserter: NewAsserter(),
		metrics: make(map[string]interface{}),
		logs:    make([]string, 0),
	}
}

func (tc *DefaultTestContext) Context() context.Context {
	return tc.ctx
}

func (tc *DefaultTestContext) Assert() *Asserter {
	return tc.asserter
}

func (tc *DefaultTestContext) Log(message string) {
	tc.logs = append(tc.logs, fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), message))
}

func (tc *DefaultTestContext) SetMetric(key string, value interface{}) {
	tc.metrics[key] = value
}

func (tc *DefaultTestContext) GetMetric(key string) interface{} {
	return tc.metrics[key]
}

// Asserter provides assertion methods for tests
type Asserter struct {
	results []AssertionResult
}

// NewAsserter creates a new asserter
func NewAsserter() *Asserter {
	return &Asserter{
		results: make([]AssertionResult, 0),
	}
}

// Equal asserts that two values are equal
func (a *Asserter) Equal(expected, actual interface{}, description string) {
	passed := expected == actual
	result := AssertionResult{
		Description: description,
		Passed:      passed,
		Expected:    expected,
		Actual:      actual,
	}

	if !passed {
		result.Error = fmt.Errorf("expected %v, got %v", expected, actual)
	}

	a.results = append(a.results, result)
}

// NoError asserts that an error is nil
func (a *Asserter) NoError(err error, description string) {
	passed := err == nil
	result := AssertionResult{
		Description: description,
		Passed:      passed,
		Expected:    nil,
		Actual:      err,
	}

	if !passed {
		result.Error = fmt.Errorf("expected no error, got: %v", err)
	}

	a.results = append(a.results, result)
}

// True asserts that a condition is true
func (a *Asserter) True(condition bool, description string) {
	result := AssertionResult{
		Description: description,
		Passed:      condition,
		Expected:    true,
		Actual:      condition,
	}

	if !condition {
		result.Error = fmt.Errorf("expected true, got false")
	}

	a.results = append(a.results, result)
}

// GetResults returns all assertion results
func (a *Asserter) GetResults() []AssertionResult {
	return a.results
}

// HasFailures returns true if any assertions failed
func (a *Asserter) HasFailures() bool {
	for _, result := range a.results {
		if !result.Passed {
			return true
		}
	}
	return false
}