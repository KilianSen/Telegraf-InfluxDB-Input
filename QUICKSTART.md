# Quick Start Guide

This guide will help you get started with the Telegraf InfluxDB3 Input Plugin quickly.

## Prerequisites

1. **Go 1.25+** - Required to build the plugin
2. **Telegraf** - The plugin runs within Telegraf
3. **InfluxDB3-core** - The data source you'll be querying

## Installation Steps

### 1. Build the Plugin

```bash
# Clone the repository
git clone https://github.com/KilianSen/Telegraf-InfluxDB-Input.git
cd Telegraf-InfluxDB-Input

# Build using Make
make build

# Or build using Go directly
go build -o telegraf-influxdb-input .
```

### 2. Install the Binary

```bash
# Install to system path
sudo make install

# Or copy manually
sudo cp telegraf-influxdb-input /usr/local/bin/
sudo chmod +x /usr/local/bin/telegraf-influxdb-input
```

### 3. Configure Environment Variables

```bash
# Copy the example environment file
cp .env.example .env

# Edit with your values
nano .env
```

Update the following values:
- `INFLUXDB_URL`: Your InfluxDB3 instance URL
- `INFLUXDB_TOKEN`: Your authentication token
- `INFLUXDB_DATABASE`: Database/bucket name
- `INFLUXDB_QUERY`: SQL query to execute

### 4. Test Standalone

```bash
# Load environment variables
export $(cat .env | xargs)

# Run the plugin
./telegraf-influxdb-input
```

You should see metrics output in InfluxDB line protocol format.

### 5. Configure Telegraf

Create a configuration file at `/etc/telegraf/telegraf.d/influxdb-input.conf`:

```toml
[[inputs.execd]]
  command = ["/usr/local/bin/telegraf-influxdb-input"]
  
  environment = [
    "INFLUXDB_URL=http://localhost:8086",
    "INFLUXDB_TOKEN=your-token-here",
    "INFLUXDB_DATABASE=mydb",
    "INFLUXDB_QUERY=SELECT * FROM metrics ORDER BY time DESC LIMIT 100"
  ]
  
  data_format = "influx"
  restart_policy = "on-failure"
  restart_delay = "10s"
```

### 6. Start Telegraf

```bash
# Start Telegraf
sudo systemctl start telegraf

# Check status
sudo systemctl status telegraf

# View logs
sudo journalctl -u telegraf -f
```

## Example Queries

### Get Recent Metrics

```sql
SELECT * FROM metrics ORDER BY time DESC LIMIT 100
```

### Time Range Query

```sql
SELECT * FROM metrics 
WHERE time >= now() - INTERVAL '1 hour'
```

### Aggregate Query

```sql
SELECT 
  time_bucket('5m', time) as bucket,
  avg(value) as avg_value,
  max(value) as max_value
FROM sensors
WHERE time >= now() - INTERVAL '1 day'
GROUP BY bucket
ORDER BY bucket DESC
```

### Specific Measurements

```sql
SELECT time, temperature, humidity, location
FROM sensors
WHERE location = 'datacenter'
  AND time >= now() - INTERVAL '15 minutes'
```

## Troubleshooting

### Plugin Won't Start

```bash
# Check if binary is executable
ls -l /usr/local/bin/telegraf-influxdb-input

# Check Telegraf logs
sudo journalctl -u telegraf -n 50
```

### Connection Errors

```bash
# Test InfluxDB connection
curl -H "Authorization: Bearer $INFLUXDB_TOKEN" \
  http://localhost:8086/health

# Verify environment variables
env | grep INFLUXDB
```

### Query Errors

```bash
# Test query directly
curl -X POST http://localhost:8086/api/v3/query_sql \
  -H "Authorization: Bearer $INFLUXDB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "database": "mydb",
    "query": "SELECT * FROM metrics LIMIT 5",
    "format": "json"
  }'
```

### Enable Debug Logging

Edit `/etc/telegraf/telegraf.conf`:

```toml
[agent]
  debug = true
```

Then restart Telegraf:

```bash
sudo systemctl restart telegraf
sudo journalctl -u telegraf -f
```

## Next Steps

- Configure outputs to send data to your desired destination
- Adjust polling interval in Telegraf agent configuration
- Create custom SQL queries for your specific data
- Set up alerting based on the collected metrics

## Support

For issues, please open a ticket on [GitHub](https://github.com/KilianSen/Telegraf-InfluxDB-Input/issues).
