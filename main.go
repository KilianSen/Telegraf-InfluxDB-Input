package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/metric"
	"github.com/influxdata/telegraf/plugins/inputs"
)

const sampleConfig = `
  ## InfluxDB3 Core instance URL
  url = "http://localhost:8181"
  
  ## API Token for authentication
  token = ""
  
  ## Organization (for InfluxDB v2/v3 compatibility)
  organization = ""
  
  ## Database/Bucket to query
  database = "telegraf"
  
  ## Query to execute for fetching data
  ## Use InfluxQL or SQL depending on your InfluxDB3 setup
  query = "SELECT * FROM metrics ORDER BY time DESC LIMIT 100"
  
  ## Polling interval (how often to check for updates)
  ## This is handled by Telegraf's interval setting
  
  ## Timeout for HTTP requests
  timeout = "5s"
  
  ## Optional TLS Config
  # tls_ca = "/etc/telegraf/ca.pem"
  # tls_cert = "/etc/telegraf/cert.pem"
  # tls_key = "/etc/telegraf/key.pem"
  # insecure_skip_verify = false
`

// InfluxDBInput represents the input plugin
type InfluxDBInput struct {
	URL                string `toml:"url"`
	Token              string `toml:"token"`
	Organization       string `toml:"organization"`
	Database           string `toml:"database"`
	Query              string `toml:"query"`
	Timeout            string `toml:"timeout"`
	TLSCA              string `toml:"tls_ca"`
	TLSCert            string `toml:"tls_cert"`
	TLSKey             string `toml:"tls_key"`
	InsecureSkipVerify bool   `toml:"insecure_skip_verify"`

	client  *http.Client
	timeout time.Duration
	Log     telegraf.Logger `toml:"-"`
}

// Description returns a short description of the plugin
func (i *InfluxDBInput) Description() string {
	return "Read metrics from InfluxDB3 Core instance"
}

// SampleConfig returns sample configuration for the plugin
func (i *InfluxDBInput) SampleConfig() string {
	return sampleConfig
}

// Init initializes the plugin
func (i *InfluxDBInput) Init() error {
	var err error
	i.timeout, err = time.ParseDuration(i.Timeout)
	if err != nil {
		i.timeout = 5 * time.Second
	}

	// Setup TLS configuration
	tlsConfig := &tls.Config{
		InsecureSkipVerify: i.InsecureSkipVerify,
	}

	// Create HTTP client
	i.client = &http.Client{
		Timeout: i.timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return nil
}

// Gather collects metrics from InfluxDB
func (i *InfluxDBInput) Gather(acc telegraf.Accumulator) error {
	ctx, cancel := context.WithTimeout(context.Background(), i.timeout)
	defer cancel()

	// Try SQL query first (InfluxDB3 Core uses SQL)
	metrics, err := i.querySQLAPI(ctx)
	if err != nil {
		i.Log.Errorf("Failed to query InfluxDB3: %v", err)
		return err
	}

	// Add metrics to accumulator
	for _, m := range metrics {
		acc.AddFields(m.Name, m.Fields, m.Tags, m.Time)
	}

	return nil
}

// querySQLAPI queries the InfluxDB3 SQL API
func (i *InfluxDBInput) querySQLAPI(ctx context.Context) ([]MetricData, error) {
	// Build the SQL query URL
	queryURL := fmt.Sprintf("%s/api/v3/query_sql", strings.TrimRight(i.URL, "/"))

	// Create request body
	requestBody := map[string]interface{}{
		"db":     i.Database,
		"q":      i.Query,
		"format": "json",
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", queryURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if i.Token != "" {
		req.Header.Set("Authorization", "Bearer "+i.Token)
	}

	// Execute request
	resp, err := i.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON response into metrics
	var result []map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to metrics
	metrics := make([]MetricData, 0, len(result))
	for _, row := range result {
		m := i.convertRowToMetric(row)
		if m != nil {
			metrics = append(metrics, *m)
		}
	}

	return metrics, nil
}

// MetricData represents a metric with its metadata
type MetricData struct {
	Name   string
	Fields map[string]interface{}
	Tags   map[string]string
	Time   time.Time
}

// convertRowToMetric converts a query result row into a metric
func (i *InfluxDBInput) convertRowToMetric(row map[string]interface{}) *MetricData {
	m := &MetricData{
		Name:   "influxdb3_query_result",
		Fields: make(map[string]interface{}),
		Tags:   make(map[string]string),
		Time:   time.Now(),
	}

	// Extract time if present
	if t, ok := row["time"]; ok {
		switch v := t.(type) {
		case string:
			if parsedTime, err := time.Parse(time.RFC3339, v); err == nil {
				m.Time = parsedTime
			}
		case float64:
			m.Time = time.Unix(int64(v), 0)
		}
		delete(row, "time")
	}

	// Extract measurement name if present
	if name, ok := row["_measurement"]; ok {
		if nameStr, ok := name.(string); ok {
			m.Name = nameStr
		}
		delete(row, "_measurement")
	}

	// Separate tags and fields
	// InfluxDB convention:
	// - String values are typically tags (metadata)
	// - Numeric, boolean, and special field values are fields (measurements)
	// - Fields starting with underscore (except _measurement) are special fields
	for key, value := range row {
		// Skip if key starts with underscore (special fields like _field, _value)
		// but still add them as fields to preserve data
		if strings.HasPrefix(key, "_") {
			if value != nil {
				m.Fields[key] = value
			}
		} else if strVal, ok := value.(string); ok {
			// String values become tags
			m.Tags[key] = strVal
		} else {
			// Numeric, boolean, and other types become fields
			m.Fields[key] = value
		}
	}

	// Ensure we have at least one field
	if len(m.Fields) == 0 {
		return nil
	}

	return m
}

// Start starts the plugin (for service inputs)
func (i *InfluxDBInput) Start(acc telegraf.Accumulator) error {
	return nil
}

// Stop stops the plugin
func (i *InfluxDBInput) Stop() {
	// Cleanup if needed
}

func init() {
	inputs.Add("influxdb_input", func() telegraf.Input {
		return &InfluxDBInput{
			URL:      "http://localhost:8181",
			Database: "control",
			Timeout:  "5s",
		}
	})
}

func main() {
	// Command line flags
	configFile := flag.String("config", "", "Configuration file path")
	flag.Parse()

	// For external plugin, we need to use Telegraf's shim
	// This allows the plugin to run as a standalone executable

	if *configFile != "" {
		fmt.Printf("Using config file: %s\n", *configFile)
	}

	// Create plugin instance
	plugin := &InfluxDBInput{
		URL:      os.Getenv("INFLUXDB_URL"),
		Token:    os.Getenv("INFLUXDB_TOKEN"),
		Database: os.Getenv("INFLUXDB_DATABASE"),
		Query:    os.Getenv("INFLUXDB_QUERY"),
	}

	// Set defaults if not provided
	if plugin.URL == "" {
		plugin.URL = "http://localhost:8181"
	}
	if plugin.Database == "" {
		plugin.Database = "control"
	}
	if plugin.Query == "" {
		plugin.Query = "SELECT * FROM opcua ORDER BY time DESC LIMIT 100"
	}

	// Initialize logger
	plugin.Log = &simpleLogger{}

	// Initialize the plugin
	if err := plugin.Init(); err != nil {
		log.Fatalf("Failed to initialize plugin: %v", err)
	}

	// Create accumulator
	acc := &simpleAccumulator{}

	// Gather metrics once (in real usage, Telegraf handles the polling)
	if err := plugin.Gather(acc); err != nil {
		log.Fatalf("Failed to gather metrics: %v", err)
	}

	// Output metrics in line protocol format
	for _, m := range acc.metrics {
		// Format as line protocol manually
		fmt.Printf("%s", formatLineProtocol(m))
	}
}

// formatLineProtocol formats a metric in InfluxDB line protocol format
func formatLineProtocol(m telegraf.Metric) string {
	var sb strings.Builder

	// Write measurement name
	sb.WriteString(m.Name())

	// Write tags
	tags := m.Tags()
	if len(tags) > 0 {
		for k, v := range tags {
			sb.WriteString(",")
			sb.WriteString(k)
			sb.WriteString("=")
			sb.WriteString(v)
		}
	}

	// Write fields
	sb.WriteString(" ")
	fields := m.Fields()
	first := true
	for k, v := range fields {
		if !first {
			sb.WriteString(",")
		}
		first = false
		sb.WriteString(k)
		sb.WriteString("=")
		switch val := v.(type) {
		case string:
			sb.WriteString(fmt.Sprintf("\"%s\"", val))
		case int, int64, int32, int16, int8:
			sb.WriteString(fmt.Sprintf("%di", val))
		case float64, float32:
			sb.WriteString(fmt.Sprintf("%f", val))
		case bool:
			sb.WriteString(fmt.Sprintf("%t", val))
		default:
			sb.WriteString(fmt.Sprintf("%v", val))
		}
	}

	// Write timestamp (in nanoseconds)
	sb.WriteString(" ")
	sb.WriteString(fmt.Sprintf("%d", m.Time().UnixNano()))
	sb.WriteString("\n")

	return sb.String()
}

// simpleLogger is a basic logger implementation for standalone execution
type simpleLogger struct{}

func (l *simpleLogger) AddAttribute(key string, value interface{}) {
	// Not implemented for simple logger
}

func (l *simpleLogger) Level() telegraf.LogLevel {
	return telegraf.Info
}

func (l *simpleLogger) Errorf(format string, args ...interface{}) {
	log.Printf("ERROR: "+format, args...)
}

func (l *simpleLogger) Error(args ...interface{}) {
	log.Print(append([]interface{}{"ERROR: "}, args...)...)
}

func (l *simpleLogger) Debugf(format string, args ...interface{}) {
	log.Printf("DEBUG: "+format, args...)
}

func (l *simpleLogger) Debug(args ...interface{}) {
	log.Print(append([]interface{}{"DEBUG: "}, args...)...)
}

func (l *simpleLogger) Warnf(format string, args ...interface{}) {
	log.Printf("WARN: "+format, args...)
}

func (l *simpleLogger) Warn(args ...interface{}) {
	log.Print(append([]interface{}{"WARN: "}, args...)...)
}

func (l *simpleLogger) Infof(format string, args ...interface{}) {
	log.Printf("INFO: "+format, args...)
}

func (l *simpleLogger) Info(args ...interface{}) {
	log.Print(append([]interface{}{"INFO: "}, args...)...)
}

func (l *simpleLogger) Trace(args ...interface{}) {
	log.Print(append([]interface{}{"TRACE: "}, args...)...)
}

func (l *simpleLogger) Tracef(format string, args ...interface{}) {
	log.Printf("TRACE: "+format, args...)
}

// simpleAccumulator is a basic accumulator implementation for standalone execution
type simpleAccumulator struct {
	metrics []telegraf.Metric
}

func (a *simpleAccumulator) AddFields(measurement string, fields map[string]interface{}, tags map[string]string, t ...time.Time) {
	timestamp := time.Now()
	if len(t) > 0 {
		timestamp = t[0]
	}

	m := metric.New(measurement, tags, fields, timestamp)
	a.metrics = append(a.metrics, m)
}

func (a *simpleAccumulator) AddGauge(measurement string, fields map[string]interface{}, tags map[string]string, t ...time.Time) {
	a.AddFields(measurement, fields, tags, t...)
}

func (a *simpleAccumulator) AddCounter(measurement string, fields map[string]interface{}, tags map[string]string, t ...time.Time) {
	a.AddFields(measurement, fields, tags, t...)
}

func (a *simpleAccumulator) AddSummary(measurement string, fields map[string]interface{}, tags map[string]string, t ...time.Time) {
	a.AddFields(measurement, fields, tags, t...)
}

func (a *simpleAccumulator) AddHistogram(measurement string, fields map[string]interface{}, tags map[string]string, t ...time.Time) {
	a.AddFields(measurement, fields, tags, t...)
}

func (a *simpleAccumulator) AddMetric(m telegraf.Metric) {
	a.metrics = append(a.metrics, m)
}

func (a *simpleAccumulator) SetPrecision(precision time.Duration) {
	// Not implemented for simple accumulator
}

func (a *simpleAccumulator) AddError(err error) {
	log.Printf("Accumulator error: %v", err)
}

func (a *simpleAccumulator) WithTracking(maxTracked int) telegraf.TrackingAccumulator {
	return nil
}
