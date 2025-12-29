# Deployment Guide

This guide covers deploying the Telegraf InfluxDB3 Input Plugin in various environments.

## Table of Contents
- [System Requirements](#system-requirements)
- [Linux Deployment](#linux-deployment)
- [Docker Deployment](#docker-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Systemd Service](#systemd-service)
- [Security Best Practices](#security-best-practices)

## System Requirements

### Minimum Requirements
- **CPU:** 1 core
- **Memory:** 256 MB
- **Disk:** 100 MB
- **Network:** Outbound HTTPS to InfluxDB3 instance

### Recommended Requirements
- **CPU:** 2 cores
- **Memory:** 512 MB
- **Disk:** 1 GB (for logs)
- **Network:** Low-latency connection to InfluxDB3

### Software Requirements
- Go 1.25+ (for building)
- Telegraf 1.30+
- Linux, macOS, or Windows

## Linux Deployment

### Ubuntu/Debian

```bash
# Install dependencies
sudo apt-get update
sudo apt-get install -y wget git golang-go

# Install Telegraf
wget https://dl.influxdata.com/telegraf/releases/telegraf_1.30.0-1_amd64.deb
sudo dpkg -i telegraf_1.30.0-1_amd64.deb

# Build and install plugin
git clone https://github.com/KilianSen/Telegraf-InfluxDB-Input.git
cd Telegraf-InfluxDB-Input
go build -o telegraf-influxdb-input .
sudo cp telegraf-influxdb-input /usr/local/bin/
sudo chmod +x /usr/local/bin/telegraf-influxdb-input

# Configure
sudo mkdir -p /etc/telegraf/telegraf.d
sudo cp telegraf-config.toml /etc/telegraf/telegraf.d/influxdb-input.conf

# Edit configuration with your values
sudo nano /etc/telegraf/telegraf.d/influxdb-input.conf

# Start Telegraf
sudo systemctl enable telegraf
sudo systemctl start telegraf
sudo systemctl status telegraf
```

### CentOS/RHEL

```bash
# Install dependencies
sudo yum update -y
sudo yum install -y wget git golang

# Install Telegraf
wget https://dl.influxdata.com/telegraf/releases/telegraf-1.30.0-1.x86_64.rpm
sudo yum localinstall -y telegraf-1.30.0-1.x86_64.rpm

# Build and install plugin
git clone https://github.com/KilianSen/Telegraf-InfluxDB-Input.git
cd Telegraf-InfluxDB-Input
go build -o telegraf-influxdb-input .
sudo cp telegraf-influxdb-input /usr/local/bin/
sudo chmod +x /usr/local/bin/telegraf-influxdb-input

# Configure
sudo mkdir -p /etc/telegraf/telegraf.d
sudo cp telegraf-config.toml /etc/telegraf/telegraf.d/influxdb-input.conf

# Edit configuration
sudo vi /etc/telegraf/telegraf.d/influxdb-input.conf

# Start Telegraf
sudo systemctl enable telegraf
sudo systemctl start telegraf
```

## Docker Deployment

### Dockerfile

Create a `Dockerfile`:

```dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy source
COPY . .

# Build plugin
RUN go build -o telegraf-influxdb-input .

# Runtime image
FROM telegraf:1.30-alpine

# Copy plugin binary
COPY --from=builder /build/telegraf-influxdb-input /usr/local/bin/

# Copy configuration
COPY telegraf-config.toml /etc/telegraf/telegraf.conf

# Ensure binary is executable
RUN chmod +x /usr/local/bin/telegraf-influxdb-input

# Run Telegraf
CMD ["telegraf"]
```

### Build and Run

```bash
# Build image
docker build -t telegraf-influxdb-input:latest .

# Run container
docker run -d \
  --name telegraf-influxdb \
  -e INFLUXDB_URL=http://influxdb:8086 \
  -e INFLUXDB_TOKEN=your-token \
  -e INFLUXDB_DATABASE=mydb \
  -e INFLUXDB_QUERY="SELECT * FROM metrics ORDER BY time DESC LIMIT 100" \
  telegraf-influxdb-input:latest
```

### Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  telegraf:
    build: .
    container_name: telegraf-influxdb-input
    restart: unless-stopped
    environment:
      - INFLUXDB_URL=http://influxdb:8086
      - INFLUXDB_TOKEN=${INFLUXDB_TOKEN}
      - INFLUXDB_DATABASE=mydb
      - INFLUXDB_QUERY=SELECT * FROM metrics ORDER BY time DESC LIMIT 100
    volumes:
      - ./telegraf-config.toml:/etc/telegraf/telegraf.conf:ro
    networks:
      - monitoring

  influxdb:
    image: influxdb:2.7
    container_name: influxdb
    restart: unless-stopped
    ports:
      - "8086:8086"
    volumes:
      - influxdb-data:/var/lib/influxdb2
    networks:
      - monitoring

networks:
  monitoring:
    driver: bridge

volumes:
  influxdb-data:
```

Run with:
```bash
docker-compose up -d
```

## Kubernetes Deployment

### ConfigMap

Create `configmap.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: telegraf-config
  namespace: monitoring
data:
  telegraf.conf: |
    [agent]
      interval = "10s"
      round_interval = true
      metric_batch_size = 1000
      
    [[inputs.execd]]
      command = ["/usr/local/bin/telegraf-influxdb-input"]
      environment = [
        "INFLUXDB_URL=${INFLUXDB_URL}",
        "INFLUXDB_TOKEN=${INFLUXDB_TOKEN}",
        "INFLUXDB_DATABASE=${INFLUXDB_DATABASE}",
        "INFLUXDB_QUERY=${INFLUXDB_QUERY}"
      ]
      data_format = "influx"
      restart_policy = "on-failure"
      
    [[outputs.influxdb_v2]]
      urls = ["${OUTPUT_INFLUXDB_URL}"]
      token = "${OUTPUT_INFLUXDB_TOKEN}"
      organization = "${OUTPUT_INFLUXDB_ORG}"
      bucket = "${OUTPUT_INFLUXDB_BUCKET}"
```

### Secret

Create `secret.yaml`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: telegraf-secrets
  namespace: monitoring
type: Opaque
stringData:
  influxdb-token: "your-input-token"
  output-influxdb-token: "your-output-token"
```

### Deployment

Create `deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: telegraf-influxdb-input
  namespace: monitoring
  labels:
    app: telegraf
spec:
  replicas: 1
  selector:
    matchLabels:
      app: telegraf
  template:
    metadata:
      labels:
        app: telegraf
    spec:
      containers:
      - name: telegraf
        image: your-registry/telegraf-influxdb-input:latest
        env:
        - name: INFLUXDB_URL
          value: "http://influxdb.monitoring.svc.cluster.local:8086"
        - name: INFLUXDB_TOKEN
          valueFrom:
            secretKeyRef:
              name: telegraf-secrets
              key: influxdb-token
        - name: INFLUXDB_DATABASE
          value: "metrics"
        - name: INFLUXDB_QUERY
          value: "SELECT * FROM metrics ORDER BY time DESC LIMIT 100"
        - name: OUTPUT_INFLUXDB_URL
          value: "http://influxdb-output.monitoring.svc.cluster.local:8086"
        - name: OUTPUT_INFLUXDB_TOKEN
          valueFrom:
            secretKeyRef:
              name: telegraf-secrets
              key: output-influxdb-token
        - name: OUTPUT_INFLUXDB_ORG
          value: "myorg"
        - name: OUTPUT_INFLUXDB_BUCKET
          value: "telegraf"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        volumeMounts:
        - name: config
          mountPath: /etc/telegraf
      volumes:
      - name: config
        configMap:
          name: telegraf-config
```

Deploy to Kubernetes:

```bash
# Create namespace
kubectl create namespace monitoring

# Apply configurations
kubectl apply -f secret.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml

# Check status
kubectl get pods -n monitoring
kubectl logs -f deployment/telegraf-influxdb-input -n monitoring
```

## Systemd Service

For running the plugin as a standalone service (without Telegraf).

Create `/etc/systemd/system/telegraf-influxdb-input.service`:

```ini
[Unit]
Description=Telegraf InfluxDB3 Input Plugin
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=telegraf
Group=telegraf
Environment="INFLUXDB_URL=http://localhost:8086"
Environment="INFLUXDB_TOKEN=your-token"
Environment="INFLUXDB_DATABASE=mydb"
Environment="INFLUXDB_QUERY=SELECT * FROM metrics ORDER BY time DESC LIMIT 100"
ExecStart=/usr/local/bin/telegraf-influxdb-input
Restart=on-failure
RestartSec=10s
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
# Create user
sudo useradd -r -s /bin/false telegraf

# Set permissions
sudo chown telegraf:telegraf /usr/local/bin/telegraf-influxdb-input

# Reload systemd
sudo systemctl daemon-reload

# Enable and start
sudo systemctl enable telegraf-influxdb-input
sudo systemctl start telegraf-influxdb-input

# Check status
sudo systemctl status telegraf-influxdb-input
```

## Security Best Practices

### 1. Token Management

**Never hardcode tokens in configuration files.**

Use environment variables:
```bash
export INFLUXDB_TOKEN=$(vault kv get -field=token secret/influxdb)
```

Or use secret management:
- HashiCorp Vault
- AWS Secrets Manager
- Azure Key Vault
- Kubernetes Secrets

### 2. TLS/SSL Configuration

Always use HTTPS in production:

```toml
environment = [
  "INFLUXDB_URL=https://influxdb.example.com:8086",
  ...
]
```

### 3. Network Security

- Use firewall rules to restrict access
- Use VPN or private networking
- Implement network segmentation
- Use security groups (cloud environments)

### 4. Least Privilege Access

Create read-only tokens for the plugin:

```bash
# InfluxDB CLI
influx auth create \
  --org myorg \
  --description "Telegraf Input Plugin Read-Only" \
  --read-bucket mydb
```

### 5. Monitoring and Alerting

Monitor the plugin:
- Log collection and analysis
- Resource usage (CPU, memory, network)
- Error rates and types
- Query performance

### 6. Update Strategy

Keep the plugin updated:

```bash
# Pull latest changes
cd Telegraf-InfluxDB-Input
git pull origin main

# Rebuild
go build -o telegraf-influxdb-input .

# Deploy
sudo cp telegraf-influxdb-input /usr/local/bin/

# Restart service
sudo systemctl restart telegraf
```

## Validation

After deployment, validate the setup:

```bash
# Check plugin is running
ps aux | grep telegraf-influxdb-input

# Check Telegraf logs
sudo journalctl -u telegraf -f

# Test connectivity
curl -H "Authorization: Bearer $INFLUXDB_TOKEN" \
  http://localhost:8086/health

# Verify metrics are being collected
# (Check your output destination)
```

## Troubleshooting Deployment

### Issue: Plugin not starting

**Check:**
- Binary exists and is executable
- Environment variables are set
- InfluxDB is accessible
- Token has correct permissions

### Issue: High resource usage

**Solutions:**
- Reduce query result size (use LIMIT)
- Increase polling interval
- Optimize queries
- Add indexes to InfluxDB

### Issue: Connection timeouts

**Solutions:**
- Increase timeout in configuration
- Check network connectivity
- Verify firewall rules
- Check InfluxDB load

### Issue: Data not appearing in output

**Check:**
- Telegraf output configuration
- Output destination connectivity
- Telegraf logs for errors
- Query returns data

## Production Checklist

- [ ] Binary built and tested
- [ ] Configuration validated
- [ ] Tokens stored securely
- [ ] TLS/SSL enabled
- [ ] Monitoring configured
- [ ] Logging configured
- [ ] Resource limits set
- [ ] Auto-restart configured
- [ ] Backup strategy defined
- [ ] Rollback plan documented
- [ ] Team trained on operations

## Support

For deployment issues:
1. Check logs: `journalctl -u telegraf -f`
2. Test manually: `./telegraf-influxdb-input`
3. Validate query: Test directly against InfluxDB
4. Open issue: [GitHub Issues](https://github.com/KilianSen/Telegraf-InfluxDB-Input/issues)
