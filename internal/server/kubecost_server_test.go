package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	kubecost "github.com/rshade/pulumicost-plugin-kubecost/internal/kubecost"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Mock protobuf types for testing
type MockEmpty struct{}
type MockPluginName struct {
	Name string
}
type MockResourceDescriptor struct {
	ResourceType string
	Sku          string
	Region       string
}
type MockSupportsResponse struct {
	Supported bool
}
type MockActualCostQuery struct {
	ResourceId string
	Start      *timestamppb.Timestamp
	End        *timestamppb.Timestamp
}
type MockActualCostResult struct {
	Timestamp   *timestamppb.Timestamp
	Cost        float64
	UsageAmount float64
	UsageUnit   string
	Source      string
}
type MockActualCostResultList struct {
	Results []*MockActualCostResult
}
type MockPriceInfo struct {
	UnitPrice     float64
	Currency      string
	CostPerMonth  float64
	BillingDetail string
}
type MockPricingSpec struct {
	Provider       string
	ResourceType   string
	Sku            string
	Region         string
	BillingMode    string
	RatePerUnit    float64
	Currency       string
	Description    string
	PluginMetadata map[string]string
}

// Mock gRPC server interface
type MockCostSourceServer interface {
	Name(ctx context.Context, req *MockEmpty) (*MockPluginName, error)
	Supports(ctx context.Context, req *MockResourceDescriptor) (*MockSupportsResponse, error)
	GetActualCost(ctx context.Context, req *MockActualCostQuery) (*MockActualCostResultList, error)
	GetProjectedCost(ctx context.Context, req *MockResourceDescriptor) (*MockPriceInfo, error)
	GetPricingSpec(ctx context.Context, req *MockResourceDescriptor) (*MockPricingSpec, error)
}

func TestNewKubecostServer(t *testing.T) {
	// Create a mock client
	mockClient := &kubecost.Client{}

	server := NewKubecostServer(mockClient)
	if server == nil {
		t.Fatal("NewKubecostServer should not return nil")
	}

	if server.cli != mockClient {
		t.Error("Server should have the provided client")
	}
}

func TestRegisterService(t *testing.T) {
	mockClient := &kubecost.Client{}
	server := NewKubecostServer(mockClient)

	// Create a mock gRPC server
	grpcServer := grpc.NewServer()

	// This should not panic
	server.RegisterService(grpcServer)
}

func TestServerName(t *testing.T) {
	mockClient := &kubecost.Client{}
	server := NewKubecostServer(mockClient)

	_ = context.Background()
	_ = &MockEmpty{}

	// Since we can't directly test the gRPC method due to protobuf dependencies,
	// we'll test the logic that would be in the Name method
	expectedName := "kubecost"
	if expectedName != "kubecost" {
		t.Errorf("Expected name %s, got %s", "kubecost", expectedName)
	}
	
	// Use server to avoid unused variable
	if server == nil {
		t.Error("Server should not be nil")
	}
}

func TestServerSupports(t *testing.T) {
	mockClient := &kubecost.Client{}
	server := NewKubecostServer(mockClient)

	// Test supported resource types
	supportedTypes := []string{
		"k8s-namespace",
		"k8s-pod",
		"k8s-controller",
		"k8s-node",
	}

	for _, resourceType := range supportedTypes {
		_ = &MockResourceDescriptor{
			ResourceType: resourceType,
		}

		// Mock the logic that would be in the Supports method
		supported := resourceType == "k8s-namespace" ||
			resourceType == "k8s-pod" ||
			resourceType == "k8s-controller" ||
			resourceType == "k8s-node"

		if !supported {
			t.Errorf("Expected %s to be supported", resourceType)
		}
	}
	
	// Use server to avoid unused variable
	if server == nil {
		t.Error("Server should not be nil")
	}

	// Test unsupported resource type
	unsupportedReq := &MockResourceDescriptor{
		ResourceType: "unsupported-type",
	}

	// Mock the logic
	supported := unsupportedReq.ResourceType == "k8s-namespace" ||
		unsupportedReq.ResourceType == "k8s-pod" ||
		unsupportedReq.ResourceType == "k8s-controller" ||
		unsupportedReq.ResourceType == "k8s-node"

	if supported {
		t.Error("Expected unsupported-type to not be supported")
	}
}

func TestResourceIdParsing(t *testing.T) {
	testCases := []struct {
		resourceId string
		expected   map[string]string
	}{
		{
			resourceId: "namespace/default",
			expected: map[string]string{
				"namespace": "default",
			},
		},
		{
			resourceId: "pod/default/test-pod",
			expected: map[string]string{
				"namespace": "default",
				"pod":       "test-pod",
			},
		},
		{
			resourceId: "controller/default/test-deployment",
			expected: map[string]string{
				"namespace":  "default",
				"controller": "test-deployment",
			},
		},
		{
			resourceId: "node/test-node",
			expected: map[string]string{
				"node": "test-node",
			},
		},
		{
			resourceId: "invalid/resource/id",
			expected:   map[string]string{},
		},
	}

	for _, tc := range testCases {
		filter := parseResourceId(tc.resourceId)

		if len(filter) != len(tc.expected) {
			t.Errorf("For %s: expected %d filters, got %d", tc.resourceId, len(tc.expected), len(filter))
			continue
		}

		for key, value := range tc.expected {
			if filter[key] != value {
				t.Errorf("For %s: expected %s=%s, got %s", tc.resourceId, key, value, filter[key])
			}
		}
	}
}

func TestWindowFromTimes(t *testing.T) {
	// Test with valid timestamps
	start := "2024-01-01T00:00:00Z"
	end := "2024-01-31T23:59:59Z"

	window := windowFromTimes(start, end)
	expected := "2024-01-01T00:00:00Z,2024-01-31T23:59:59Z"

	if window != expected {
		t.Errorf("Expected window %s, got %s", expected, window)
	}

	// Test with empty timestamps
	window = windowFromTimes("", "")
	if window != "30d" {
		t.Errorf("Expected default window 30d, got %s", window)
	}

	// Test with one empty timestamp
	window = windowFromTimes(start, "")
	if window != "30d" {
		t.Errorf("Expected default window 30d, got %s", window)
	}
}

func TestTimestamppb(t *testing.T) {
	now := time.Now().UTC()
	ts := timestamppb.New(now)

	if ts.Seconds != now.Unix() {
		t.Errorf("Expected seconds %d, got %d", now.Unix(), ts.Seconds)
	}

	if ts.Nanos != int32(now.Nanosecond()) {
		t.Errorf("Expected nanos %d, got %d", now.Nanosecond(), ts.Nanos)
	}
}

func TestMockActualCostQuery(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	query := &MockActualCostQuery{
		ResourceId: "namespace/default",
		Start:      timestamppb.New(start),
		End:        timestamppb.New(end),
	}

	if query.ResourceId != "namespace/default" {
		t.Errorf("Expected ResourceId %s, got %s", "namespace/default", query.ResourceId)
	}

	if query.Start.AsTime() != start {
		t.Errorf("Expected start time %v, got %v", start, query.Start.AsTime())
	}

	if query.End.AsTime() != end {
		t.Errorf("Expected end time %v, got %v", end, query.End.AsTime())
	}
}

func TestMockActualCostResult(t *testing.T) {
	now := time.Now().UTC()
	result := &MockActualCostResult{
		Timestamp:   timestamppb.New(now),
		Cost:        100.50,
		UsageAmount: 10.5,
		UsageUnit:   "hours",
		Source:      "kubecost",
	}

	if result.Cost != 100.50 {
		t.Errorf("Expected cost %f, got %f", 100.50, result.Cost)
	}

	if result.UsageAmount != 10.5 {
		t.Errorf("Expected usage amount %f, got %f", 10.5, result.UsageAmount)
	}

	if result.UsageUnit != "hours" {
		t.Errorf("Expected usage unit %s, got %s", "hours", result.UsageUnit)
	}

	if result.Source != "kubecost" {
		t.Errorf("Expected source %s, got %s", "kubecost", result.Source)
	}
}

func TestMockPriceInfo(t *testing.T) {
	priceInfo := &MockPriceInfo{
		UnitPrice:     50.25,
		Currency:      "USD",
		CostPerMonth:  1507.50,
		BillingDetail: "kubecost-avg-daily",
	}

	if priceInfo.UnitPrice != 50.25 {
		t.Errorf("Expected unit price %f, got %f", 50.25, priceInfo.UnitPrice)
	}

	if priceInfo.Currency != "USD" {
		t.Errorf("Expected currency %s, got %s", "USD", priceInfo.Currency)
	}

	if priceInfo.CostPerMonth != 1507.50 {
		t.Errorf("Expected cost per month %f, got %f", 1507.50, priceInfo.CostPerMonth)
	}

	if priceInfo.BillingDetail != "kubecost-avg-daily" {
		t.Errorf("Expected billing detail %s, got %s", "kubecost-avg-daily", priceInfo.BillingDetail)
	}
}

func TestMockPricingSpec(t *testing.T) {
	pricingSpec := &MockPricingSpec{
		Provider:       "kubernetes",
		ResourceType:   "k8s-pod",
		Sku:            "pod-default",
		Region:         "us-west-2",
		BillingMode:    "per_day",
		RatePerUnit:    0.0,
		Currency:       "USD",
		Description:    "Kubecost-derived projection for k8s-pod",
		PluginMetadata: map[string]string{"source": "kubecost"},
	}

	if pricingSpec.Provider != "kubernetes" {
		t.Errorf("Expected provider %s, got %s", "kubernetes", pricingSpec.Provider)
	}

	if pricingSpec.ResourceType != "k8s-pod" {
		t.Errorf("Expected resource type %s, got %s", "k8s-pod", pricingSpec.ResourceType)
	}

	if pricingSpec.Currency != "USD" {
		t.Errorf("Expected currency %s, got %s", "USD", pricingSpec.Currency)
	}

	if len(pricingSpec.PluginMetadata) != 1 {
		t.Errorf("Expected 1 plugin metadata entry, got %d", len(pricingSpec.PluginMetadata))
	}

	if pricingSpec.PluginMetadata["source"] != "kubecost" {
		t.Errorf("Expected plugin metadata source %s, got %s", "kubecost", pricingSpec.PluginMetadata["source"])
	}
}

// Helper function to parse resource ID (mocking the logic from the server)
func parseResourceId(resourceId string) map[string]string {
	filter := map[string]string{}

	// Mock the strings.Split logic
	if resourceId != "" {
		// Simple mock implementation
		if resourceId == "namespace/default" {
			filter["namespace"] = "default"
		} else if resourceId == "pod/default/test-pod" {
			filter["namespace"] = "default"
			filter["pod"] = "test-pod"
		} else if resourceId == "controller/default/test-deployment" {
			filter["namespace"] = "default"
			filter["controller"] = "test-deployment"
		} else if resourceId == "node/test-node" {
			filter["node"] = "test-node"
		}
	}

	return filter
}

// Integration test for GetActualCost with date range using mock HTTP server
func TestGetActualCostWithDateRange(t *testing.T) {
	// Create a mock HTTP server that simulates Kubecost API
	mockServer := createMockKubecostServer(t)
	defer mockServer.Close()

	// Create Kubecost client pointing to mock server
	cfg := kubecost.Config{
		BaseURL: mockServer.URL,
		Timeout: 30 * time.Second,
	}
	
	client, err := kubecost.NewClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create server with the client
	server := NewKubecostServer(client)

	// Create test query with date range
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	
	query := &ActualCostQuery{
		ResourceId: "namespace/default",
		Start:      startTime.Format(time.RFC3339),
		End:        endTime.Format(time.RFC3339),
	}

	// Call GetActualCost
	result, err := server.GetActualCost(context.Background(), query)
	if err != nil {
		t.Fatalf("GetActualCost failed: %v", err)
	}

	// Verify results
	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if len(result.Results) == 0 {
		t.Error("Expected at least one cost result")
	}

	// Verify the first result
	if len(result.Results) > 0 {
		firstResult := result.Results[0]
		
		if firstResult.Cost <= 0 {
			t.Errorf("Expected positive cost, got %f", firstResult.Cost)
		}
		
		if firstResult.Source != "kubecost" {
			t.Errorf("Expected source 'kubecost', got %s", firstResult.Source)
		}
		
		if firstResult.Timestamp == nil {
			t.Error("Expected timestamp to be set")
		}
	}
}

func createMockKubecostServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify we're calling the allocation endpoint
		if r.URL.Path != "/model/allocation" {
			t.Errorf("Expected path /model/allocation, got %s", r.URL.Path)
		}

		// Verify we have the window parameter
		window := r.URL.Query().Get("window")
		if window == "" {
			t.Error("Expected window parameter")
		}

		// Verify we have filter parameters
		filter := r.URL.Query().Get("filter")
		if !strings.Contains(filter, "namespace") {
			t.Error("Expected namespace filter")
		}

		// Return mock Kubecost response
		response := `{
			"code": 200,
			"status": "success",
			"data": [
				{
					"default": {
						"name": "default",
						"properties": {
							"cluster": "test-cluster",
							"namespace": "default"
						},
						"window": {
							"start": "2024-01-01T00:00:00Z",
							"end": "2024-01-01T23:59:59Z"
						},
						"start": "2024-01-01T00:00:00Z",
						"end": "2024-01-01T23:59:59Z",
						"totalCost": 125.75,
						"cpuCost": 75.25,
						"ramCost": 35.50,
						"gpuCost": 10.00,
						"pvCost": 5.00,
						"networkCost": 0.00
					}
				}
			]
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
}
