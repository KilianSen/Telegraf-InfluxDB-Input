# Telegraf InfluxDB3 Input Plugin

An external input plugin for Telegraf that connects to an InfluxDB3-core instance and polls for data updates. This plugin allows you to read metrics from an InfluxDB3 database and forward them to other outputs through Telegraf.

## Features

- 🔌 External plugin architecture for easy deployment
- 🔄 Continuous polling for data updates
- 🔐 Token-based authentication support
- 🔍 SQL query support for InfluxDB3
- ⚙️ Configurable polling intervals
- 🛡️ TLS/SSL support
- 📊 Automatic metric conversion to Telegraf format
- ✨ **Smart deduplication - only propagates new metrics** (prevents duplicate data)

## Installation

### Prerequisites

- Go 1.25 or later
- Telegraf installed on your system
- Access to an InfluxDB3-core instance

### Build from Source

```bash
# Clone the repository
git clone https://github.com/KilianSen/Telegraf-InfluxDB-Input.git
cd Telegraf-InfluxDB-Input

# Build the plugin
go build -o telegraf-influxdb-input .

# Make it executable
chmod +x telegraf-influxdb-input
```

### Install the Binary

Move the compiled binary to a suitable location:

```bash
sudo mv telegraf-influxdb-input /usr/local/bin/
```

## Configuration

### Using Environment Variables

The plugin can be configured using environment variables:

- `INFLUXDB_URL` - InfluxDB3 instance URL (default: `http://localhost:8086`)
- `INFLUXDB_TOKEN` - API token for authentication
- `INFLUXDB_DATABASE` - Database/bucket to query (default: `telegraf`)
- `INFLUXDB_QUERY` - SQL query to execute (default: `SELECT * FROM metrics ORDER BY time DESC LIMIT 100`)

### Telegraf Configuration

Add the following to your Telegraf configuration file (usually `/etc/telegraf/telegraf.conf`):

```toml
[[inputs.execd]]
  ## Command to run the external plugin
  command = ["/usr/local/bin/telegraf-influxdb-input"]
  
  ## Environment variables to pass to the plugin
  environment = [
    "INFLUXDB_URL=http://localhost:8086",
    "INFLUXDB_TOKEN=your-token-here",
    "INFLUXDB_DATABASE=mydb",
    "INFLUXDB_QUERY=SELECT * FROM metrics ORDER BY time DESC LIMIT 100"
  ]
  
  ## Data format to consume
  data_format = "influx"
  
  ## Restart policy (always, on-failure, never)
  restart_policy = "on-failure"
  
  ## Restart delay
  restart_delay = "10s"
```

### Sample Configuration

A complete sample configuration file is provided in `telegraf-config.toml`. You can use it as a starting point:

```bash
cp telegraf-config.toml /etc/telegraf/telegraf.d/influxdb-input.conf
```

## Usage

### Running Standalone

You can run the plugin standalone for testing:

```bash
export INFLUXDB_URL="http://localhost:8086"
export INFLUXDB_TOKEN="your-token-here"
export INFLUXDB_DATABASE="mydb"
export INFLUXDB_QUERY="SELECT * FROM metrics ORDER BY time DESC LIMIT 100"

./telegraf-influxdb-input
```

This will output metrics in InfluxDB line protocol format.

### Running with Telegraf

Start Telegraf with your configuration:

```bash
telegraf --config /etc/telegraf/telegraf.conf
```

Telegraf will automatically start the plugin and collect metrics at the configured interval.

## Query Examples

### SQL Queries for InfluxDB3

```sql
-- Get recent metrics
SELECT * FROM metrics ORDER BY time DESC LIMIT 100

-- Get metrics from a specific time range
SELECT * FROM metrics WHERE time >= now() - INTERVAL '1 hour'

-- Get specific measurements
SELECT time, temperature, humidity FROM sensors WHERE location = 'office'

-- Aggregate data
SELECT 
  time_bucket('5m', time) as bucket,
  avg(value) as avg_value
FROM metrics
WHERE time >= now() - INTERVAL '1 day'
GROUP BY bucket
ORDER BY bucket DESC
```

## Architecture

The plugin works as follows:

1. **Initialization**: Sets up HTTP client with TLS configuration and initializes metric tracking
2. **Polling**: Telegraf calls the plugin at regular intervals (configured in `agent.interval`)
3. **Query Execution**: Plugin executes SQL query against InfluxDB3 API
4. **Data Transformation**: Converts query results to Telegraf metrics
5. **Deduplication**: Checks each metric against seen metrics cache (enabled by default)
6. **Output**: Returns only new metrics in InfluxDB line protocol format
7. **Cleanup**: Periodically removes old entries from the tracking cache

### Metric Deduplication

By default, the plugin tracks metrics that have been propagated and only forwards new ones. This prevents duplicate data when polling overlapping time windows.

**How it works:**
- Each metric is uniquely identified by: measurement name + tags + timestamp
- A configurable in-memory cache tracks seen metrics
- Old entries are automatically cleaned up based on `metric_tracking_window`
- Cache size is limited by `max_tracked_metrics` to prevent memory issues

**Configuration options:**
```toml
## Enable/disable deduplication (default: true)
track_new_metrics_only = true

## Maximum metrics to track (default: 10000)
max_tracked_metrics = 10000

## How long to remember metrics (default: 1h)
metric_tracking_window = "1h"
```

To disable deduplication and forward all metrics:
```toml
track_new_metrics_only = false
```

## Security Considerations

- Always use HTTPS in production environments
- Store tokens in environment variables or secure vaults, not in config files
- Use read-only tokens when possible
- Enable TLS verification (`insecure_skip_verify = false`)
- Limit query results with `LIMIT` clause to prevent memory issues

## Troubleshooting

### Plugin not starting

Check Telegraf logs:
```bash
journalctl -u telegraf -f
```

### Connection errors

Verify InfluxDB3 is accessible:
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8086/health
```

### Query errors

Test your query directly:
```bash
curl -X POST http://localhost:8086/api/v3/query_sql \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"database":"mydb","query":"SELECT * FROM metrics LIMIT 10","format":"json"}'
```

### Debug mode

Enable debug logging in Telegraf:
```toml
[agent]
  debug = true
```

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o telegraf-influxdb-input .
```

### Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Links

- [Telegraf Documentation](https://docs.influxdata.com/telegraf/)
- [InfluxDB3 Core](https://github.com/influxdata/influxdb)
- [External Plugin Documentation](https://github.com/influxdata/telegraf/blob/master/docs/EXTERNAL_PLUGINS.md)

## Support

For issues and questions:
- Open an issue on [GitHub](https://github.com/KilianSen/Telegraf-InfluxDB-Input/issues)
- Check existing issues for similar problems

## Changelog

### v1.0.0 (Initial Release)
- Initial implementation with SQL query support
- Token-based authentication
- Configurable polling
- TLS support
- Automatic metric conversion
