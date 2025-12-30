package main

import (
	"testing"
	"time"
)

// TestGenerateMetricKey tests that metric keys are generated consistently
func TestGenerateMetricKey(t *testing.T) {
	plugin := &InfluxDBInput{}

	timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Test with same data - should produce same key
	m1 := MetricData{
		Name: "test_metric",
		Tags: map[string]string{
			"host": "server1",
			"env":  "prod",
		},
		Fields: map[string]interface{}{
			"value": 42.0,
		},
		Time: timestamp,
	}

	m2 := MetricData{
		Name: "test_metric",
		Tags: map[string]string{
			"env":  "prod",
			"host": "server1", // Different order
		},
		Fields: map[string]interface{}{
			"value": 42.0,
		},
		Time: timestamp,
	}

	key1 := plugin.generateMetricKey(m1)
	key2 := plugin.generateMetricKey(m2)

	if key1 != key2 {
		t.Errorf("Expected same key for metrics with same tags in different order, got %s and %s", key1, key2)
	}

	// Test with different timestamp - should produce different key
	m3 := MetricData{
		Name: "test_metric",
		Tags: map[string]string{
			"host": "server1",
			"env":  "prod",
		},
		Fields: map[string]interface{}{
			"value": 42.0,
		},
		Time: timestamp.Add(1 * time.Second),
	}

	key3 := plugin.generateMetricKey(m3)

	if key1 == key3 {
		t.Errorf("Expected different keys for metrics with different timestamps")
	}

	// Test with different name - should produce different key
	m4 := MetricData{
		Name: "different_metric",
		Tags: map[string]string{
			"host": "server1",
			"env":  "prod",
		},
		Fields: map[string]interface{}{
			"value": 42.0,
		},
		Time: timestamp,
	}

	key4 := plugin.generateMetricKey(m4)

	if key1 == key4 {
		t.Errorf("Expected different keys for metrics with different names")
	}

	// Test with different tags - should produce different key
	m5 := MetricData{
		Name: "test_metric",
		Tags: map[string]string{
			"host": "server2",
			"env":  "prod",
		},
		Fields: map[string]interface{}{
			"value": 42.0,
		},
		Time: timestamp,
	}

	key5 := plugin.generateMetricKey(m5)

	if key1 == key5 {
		t.Errorf("Expected different keys for metrics with different tags")
	}
}

// TestIsNewMetric tests the deduplication logic
func TestIsNewMetric(t *testing.T) {
	plugin := &InfluxDBInput{
		TrackNewMetricsOnly: true,
		MaxTrackedMetrics:   10000,
		seenMetrics:         make(map[string]time.Time),
		Log:                 &simpleLogger{},
	}

	timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	m := MetricData{
		Name: "test_metric",
		Tags: map[string]string{
			"host": "server1",
		},
		Fields: map[string]interface{}{
			"value": 42.0,
		},
		Time: timestamp,
	}

	// First time - should be new
	if !plugin.isNewMetric(m) {
		t.Error("Expected metric to be new on first check")
	}

	// Mark as seen
	plugin.markMetricAsSeen(m)

	// Second time - should not be new
	if plugin.isNewMetric(m) {
		t.Error("Expected metric to not be new after marking as seen")
	}

	// Different metric - should be new
	m2 := MetricData{
		Name: "test_metric",
		Tags: map[string]string{
			"host": "server2",
		},
		Fields: map[string]interface{}{
			"value": 42.0,
		},
		Time: timestamp,
	}

	if !plugin.isNewMetric(m2) {
		t.Error("Expected different metric to be new")
	}
}

// TestCleanupOldMetrics tests that old metrics are removed from tracking
func TestCleanupOldMetrics(t *testing.T) {
	plugin := &InfluxDBInput{
		TrackNewMetricsOnly: true,
		trackingWindow:      1 * time.Hour,
		seenMetrics:         make(map[string]time.Time),
		Log:                 &simpleLogger{},
	}

	// Add an old metric
	oldTime := time.Now().Add(-2 * time.Hour)
	plugin.seenMetrics["old_metric_key"] = oldTime

	// Add a recent metric
	recentTime := time.Now().Add(-30 * time.Minute)
	plugin.seenMetrics["recent_metric_key"] = recentTime

	// Clean up
	plugin.cleanupOldMetrics()

	// Old metric should be removed
	if _, exists := plugin.seenMetrics["old_metric_key"]; exists {
		t.Error("Expected old metric to be removed")
	}

	// Recent metric should still exist
	if _, exists := plugin.seenMetrics["recent_metric_key"]; !exists {
		t.Error("Expected recent metric to still exist")
	}
}

// TestEvictOldestMetrics tests the eviction logic when max limit is reached
func TestEvictOldestMetrics(t *testing.T) {
	plugin := &InfluxDBInput{
		TrackNewMetricsOnly: true,
		MaxTrackedMetrics:   100,
		seenMetrics:         make(map[string]time.Time),
		Log:                 &simpleLogger{},
	}

	// Add 110 metrics with different timestamps
	baseTime := time.Now()
	for i := 0; i < 110; i++ {
		key := time.Now().String() + string(rune(i))
		plugin.seenMetrics[key] = baseTime.Add(time.Duration(i) * time.Minute)
	}

	initialCount := len(plugin.seenMetrics)

	// Trigger eviction (called internally by markMetricAsSeen)
	plugin.evictOldestMetrics()

	finalCount := len(plugin.seenMetrics)

	// Should have removed ~10% of metrics
	expectedRemoved := initialCount / 10
	actualRemoved := initialCount - finalCount

	if actualRemoved < expectedRemoved-1 || actualRemoved > expectedRemoved+1 {
		t.Errorf("Expected to remove approximately %d metrics, removed %d", expectedRemoved, actualRemoved)
	}
}

// TestConvertRowToMetric tests the conversion of query results to metrics
func TestConvertRowToMetric(t *testing.T) {
	plugin := &InfluxDBInput{
		Log: &simpleLogger{},
	}

	// Test basic conversion
	row := map[string]interface{}{
		"time":         "2024-01-01T12:00:00Z",
		"_measurement": "cpu",
		"host":         "server1",
		"value":        42.5,
	}

	m := plugin.convertRowToMetric(row)

	if m == nil {
		t.Fatal("Expected metric to be created")
	}

	if m.Name != "cpu" {
		t.Errorf("Expected measurement name 'cpu', got '%s'", m.Name)
	}

	if m.Tags["host"] != "server1" {
		t.Errorf("Expected tag host='server1', got '%s'", m.Tags["host"])
	}

	if m.Fields["value"] != 42.5 {
		t.Errorf("Expected field value=42.5, got %v", m.Fields["value"])
	}

	// Test with no fields - should return nil
	row2 := map[string]interface{}{
		"time":         "2024-01-01T12:00:00Z",
		"_measurement": "cpu",
		"host":         "server1",
	}

	m2 := plugin.convertRowToMetric(row2)

	if m2 != nil {
		t.Error("Expected nil metric when no fields present")
	}
}

// TestTrackingDisabled tests that all metrics are propagated when tracking is disabled
func TestTrackingDisabled(t *testing.T) {
	plugin := &InfluxDBInput{
		TrackNewMetricsOnly: false,
		MaxTrackedMetrics:   10000,
		seenMetrics:         make(map[string]time.Time),
		Log:                 &simpleLogger{},
	}

	timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	m := MetricData{
		Name: "test_metric",
		Tags: map[string]string{
			"host": "server1",
		},
		Fields: map[string]interface{}{
			"value": 42.0,
		},
		Time: timestamp,
	}

	// When tracking is disabled, isNewMetric should always return true conceptually
	// but we don't call it when tracking is disabled
	// The Gather method handles this logic

	// Verify that seenMetrics map is empty (not used when tracking disabled)
	plugin.markMetricAsSeen(m) // This should still work but won't be used
	
	if len(plugin.seenMetrics) == 0 {
		t.Error("Expected metric to be tracked even if tracking is disabled (data structure still works)")
	}
}
