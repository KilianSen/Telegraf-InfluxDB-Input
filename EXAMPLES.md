# Example Configurations and Use Cases

This document provides various example configurations for different use cases.

## Example 1: Basic Metric Polling

Poll recent metrics from a database every 30 seconds.

**Telegraf Configuration:**
```toml
[agent]
  interval = "30s"

[[inputs.execd]]
  command = ["/usr/local/bin/telegraf-influxdb-input"]
  
  environment = [
    "INFLUXDB_URL=http://localhost:8086",
    "INFLUXDB_TOKEN=mytoken123",
    "INFLUXDB_DATABASE=production",
    "INFLUXDB_QUERY=SELECT * FROM system_metrics ORDER BY time DESC LIMIT 50"
  ]
  
  data_format = "influx"

[[outputs.influxdb_v2]]
  urls = ["http://influx-target:8086"]
  token = "output-token"
  organization = "myorg"
  bucket = "aggregated"
```

## Example 2: Time-Based Incremental Updates with Deduplication

Query overlapping time windows without worrying about duplicates (deduplication is enabled by default).

**Telegraf Configuration:**
```toml
[[inputs.execd]]
  command = ["/usr/local/bin/telegraf-influxdb-input"]
  
  environment = [
    "INFLUXDB_URL=http://localhost:8086",
    "INFLUXDB_TOKEN=mytoken123",
    "INFLUXDB_DATABASE=events",
    # Query with overlap to ensure no data is missed
    "INFLUXDB_QUERY=SELECT * FROM events WHERE time >= now() - INTERVAL '2 minutes' ORDER BY time DESC"
  ]
  
  data_format = "influx"
```

**Note:** With `interval = "1m"` in agent config and a 2-minute query window, there's overlap between polls. The plugin automatically deduplicates metrics so each event is only forwarded once.

## Example 3: Multi-Measurement Aggregation

Aggregate data from multiple measurements.

**Query:**
```sql
SELECT 
  'aggregated' as _measurement,
  time_bucket('10m', time) as time,
  host,
  avg(cpu_usage) as avg_cpu,
  avg(mem_usage) as avg_mem,
  max(cpu_usage) as max_cpu,
  max(mem_usage) as max_mem
FROM system_stats
WHERE time >= now() - INTERVAL '30 minutes'
GROUP BY time_bucket('10m', time), host
ORDER BY time DESC
```

**Telegraf Configuration:**
```toml
[[inputs.execd]]
  command = ["/usr/local/bin/telegraf-influxdb-input"]
  
  environment = [
    "INFLUXDB_URL=http://localhost:8086",
    "INFLUXDB_TOKEN=mytoken123",
    "INFLUXDB_DATABASE=metrics",
    "INFLUXDB_QUERY=SELECT 'aggregated' as _measurement, time_bucket('10m', time) as time, host, avg(cpu_usage) as avg_cpu, avg(mem_usage) as avg_mem FROM system_stats WHERE time >= now() - INTERVAL '30 minutes' GROUP BY time_bucket('10m', time), host ORDER BY time DESC"
  ]
  
  data_format = "influx"
```

## Example 4: Cross-Database Monitoring

Monitor metrics from one InfluxDB instance and forward to another.

**Source InfluxDB:** Development environment metrics
**Target InfluxDB:** Production monitoring dashboard

**Telegraf Configuration:**
```toml
# Read from dev environment
[[inputs.execd]]
  command = ["/usr/local/bin/telegraf-influxdb-input"]
  
  environment = [
    "INFLUXDB_URL=http://dev-influx:8086",
    "INFLUXDB_TOKEN=dev-token",
    "INFLUXDB_DATABASE=dev_metrics",
    "INFLUXDB_QUERY=SELECT * FROM app_performance WHERE time >= now() - INTERVAL '5 minutes'"
  ]
  
  data_format = "influx"
  
  [inputs.execd.tags]
    environment = "development"

# Write to production monitoring
[[outputs.influxdb_v2]]
  urls = ["http://prod-monitoring:8086"]
  token = "prod-token"
  organization = "myorg"
  bucket = "cross_env_metrics"
```

## Example 5: Filtered Sensor Data

Pull only specific sensor readings based on conditions.

**Query:**
```sql
SELECT 
  time,
  sensor_id,
  temperature,
  humidity,
  location
FROM sensors
WHERE 
  time >= now() - INTERVAL '2 minutes'
  AND location IN ('datacenter-1', 'datacenter-2')
  AND (temperature > 25 OR humidity > 70)
ORDER BY time DESC
```

**Telegraf Configuration:**
```toml
[[inputs.execd]]
  command = ["/usr/local/bin/telegraf-influxdb-input"]
  
  environment = [
    "INFLUXDB_URL=http://localhost:8086",
    "INFLUXDB_TOKEN=mytoken123",
    "INFLUXDB_DATABASE=iot",
    "INFLUXDB_QUERY=SELECT time, sensor_id, temperature, humidity, location FROM sensors WHERE time >= now() - INTERVAL '2 minutes' AND location IN ('datacenter-1', 'datacenter-2') AND (temperature > 25 OR humidity > 70) ORDER BY time DESC"
  ]
  
  data_format = "influx"
  
  [inputs.execd.tags]
    data_source = "iot_sensors"
    alert_level = "warning"
```

## Example 6: High-Frequency Polling with Custom Deduplication Settings

Poll data every 5 seconds for real-time monitoring with optimized deduplication settings.

**Telegraf Configuration:**
```toml
[agent]
  interval = "5s"
  flush_interval = "5s"

[[inputs.execd]]
  command = ["/usr/local/bin/telegraf-influxdb-input"]
  
  environment = [
    "INFLUXDB_URL=http://localhost:8086",
    "INFLUXDB_TOKEN=mytoken123",
    "INFLUXDB_DATABASE=realtime",
    # Query with 2x overlap for safety
    "INFLUXDB_QUERY=SELECT * FROM live_metrics WHERE time >= now() - INTERVAL '10 seconds' ORDER BY time DESC LIMIT 100"
  ]
  
  data_format = "influx"
  restart_policy = "always"
```

**Note:** High-frequency polling with overlapping windows ensures no data loss. The default deduplication settings handle the overlap efficiently. For very high volumes, consider adjusting `max_tracked_metrics`.

## Example 7: Secure Connection with TLS

Connect to InfluxDB3 over HTTPS with certificate verification.

**Telegraf Configuration:**
```toml
[[inputs.execd]]
  command = ["/usr/local/bin/telegraf-influxdb-input"]
  
  environment = [
    "INFLUXDB_URL=https://secure-influx.example.com:8086",
    "INFLUXDB_TOKEN=secure-token",
    "INFLUXDB_DATABASE=secure_data",
    "INFLUXDB_QUERY=SELECT * FROM secure_metrics ORDER BY time DESC LIMIT 50"
  ]
  
  data_format = "influx"
```

**Note:** For production, store tokens in environment variables or secret management systems, not in config files.

## Example 8: Multiple Input Sources

Run multiple instances of the plugin querying different databases.

**Telegraf Configuration:**
```toml
# First input - Application metrics
[[inputs.execd]]
  command = ["/usr/local/bin/telegraf-influxdb-input"]
  
  environment = [
    "INFLUXDB_URL=http://app-db:8086",
    "INFLUXDB_TOKEN=app-token",
    "INFLUXDB_DATABASE=applications",
    "INFLUXDB_QUERY=SELECT * FROM app_metrics WHERE time >= now() - INTERVAL '1 minute'"
  ]
  
  data_format = "influx"
  name_suffix = "_app"

# Second input - Infrastructure metrics
[[inputs.execd]]
  command = ["/usr/local/bin/telegraf-influxdb-input"]
  
  environment = [
    "INFLUXDB_URL=http://infra-db:8086",
    "INFLUXDB_TOKEN=infra-token",
    "INFLUXDB_DATABASE=infrastructure",
    "INFLUXDB_QUERY=SELECT * FROM infra_metrics WHERE time >= now() - INTERVAL '1 minute'"
  ]
  
  data_format = "influx"
  name_suffix = "_infra"

# Combined output
[[outputs.influxdb_v2]]
  urls = ["http://central-monitoring:8086"]
  token = "central-token"
  organization = "myorg"
  bucket = "unified_metrics"
```

## Example 9: Data Transformation with Tags

Add contextual tags to metrics during collection.

**Telegraf Configuration:**
```toml
[[inputs.execd]]
  command = ["/usr/local/bin/telegraf-influxdb-input"]
  
  environment = [
    "INFLUXDB_URL=http://localhost:8086",
    "INFLUXDB_TOKEN=mytoken123",
    "INFLUXDB_DATABASE=raw_data",
    "INFLUXDB_QUERY=SELECT * FROM raw_metrics WHERE time >= now() - INTERVAL '1 minute'"
  ]
  
  data_format = "influx"
  
  [inputs.execd.tags]
    source = "influxdb3"
    region = "us-east-1"
    datacenter = "dc1"
    processed = "true"
```

## Example 10: Long-Running Historical Query

Query larger datasets with extended timeout.

**Telegraf Configuration:**
```toml
[agent]
  interval = "5m"

[[inputs.execd]]
  command = ["/usr/local/bin/telegraf-influxdb-input"]
  
  environment = [
    "INFLUXDB_URL=http://localhost:8086",
    "INFLUXDB_TOKEN=mytoken123",
    "INFLUXDB_DATABASE=historical",
    "INFLUXDB_QUERY=SELECT time_bucket('1h', time) as time, avg(value) as avg_value FROM metrics WHERE time >= now() - INTERVAL '24 hours' GROUP BY time_bucket('1h', time) ORDER BY time DESC"
  ]
  
  data_format = "influx"
  restart_policy = "on-failure"
```

## Environment-Specific Configurations

### Development
```bash
INFLUXDB_URL=http://localhost:8086
INFLUXDB_TOKEN=dev-token
INFLUXDB_DATABASE=dev
INFLUXDB_QUERY=SELECT * FROM metrics ORDER BY time DESC LIMIT 10
```

### Staging
```bash
INFLUXDB_URL=http://staging-influx.internal:8086
INFLUXDB_TOKEN=staging-token
INFLUXDB_DATABASE=staging
INFLUXDB_QUERY=SELECT * FROM metrics WHERE time >= now() - INTERVAL '5 minutes'
```

### Production
```bash
INFLUXDB_URL=https://influx.example.com:8086
INFLUXDB_TOKEN=${VAULT_INFLUX_TOKEN}
INFLUXDB_DATABASE=production
INFLUXDB_QUERY=SELECT * FROM metrics WHERE time >= now() - INTERVAL '1 minute' LIMIT 500
```

## Best Practices

1. **Limit Query Results**: Always use `LIMIT` to prevent memory issues
2. **Use Time Windows**: Query only recent data with `WHERE time >= now() - INTERVAL`
3. **Use Overlapping Windows**: Query with overlap (e.g., 2-minute window with 1-minute interval) to ensure no data is missed - deduplication handles duplicates automatically
4. **Index Your Queries**: Ensure InfluxDB3 has proper indexes for performance
5. **Monitor Resource Usage**: Watch CPU and memory when polling frequently
6. **Use Appropriate Intervals**: Match Telegraf interval to data freshness needs
7. **Secure Tokens**: Use environment variables or secrets management
8. **Test Queries First**: Validate SQL queries directly before adding to Telegraf
9. **Add Contextual Tags**: Use Telegraf tags to add metadata to metrics
10. **Handle Errors Gracefully**: Use `restart_policy = "on-failure"` for resilience
11. **Log and Monitor**: Enable debug logging during initial setup
12. **Configure Deduplication**: Adjust `max_tracked_metrics` and `metric_tracking_window` based on your data volume and polling frequency
