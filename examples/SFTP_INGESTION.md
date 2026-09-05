# SFTP File Ingestion Example (R21.4)

This example demonstrates real SFTP file polling using dim's file source adapter. Files are polled from a remote SFTP server, optionally parsed, enriched with metadata, and written to a local output directory.

## Quick Start

### Prerequisites

- `sftp-ingestion.yaml` route configuration
- Environment variable set: `SFTP_PASSWORD`

### Run with Remote SFTP

```bash
# Set SFTP credentials
export SFTP_PASSWORD="your-password"

# Run the route
dimctl run examples/sftp-ingestion.yaml
```

Files will be polled from the remote server every 5 minutes and written to `output/sftp-ingested/`.

## Testing with Docker SFTP

For local testing without a real SFTP server, use the included Docker SFTP test container.

### Step 1: Start SFTP Server

```bash
# Start the test SFTP server (atmoz/sftp image)
docker-compose -f deploy/docker-compose.sftp.yml up -d

# Verify it's running
docker ps | grep sftp
```

The test server runs on:
- **Host**: localhost
- **Port**: 2222
- **Username**: testuser
- **Password**: testpass

### Step 2: Create Test Files

```bash
# Connect to SFTP server
sftp -o StrictHostKeyChecking=no -P 2222 testuser@localhost

# At sftp prompt:
cd upload
put /path/to/test-file.json
put /path/to/another-file.json
exit
```

Or use automated upload:

```bash
# Create test JSON files
mkdir -p /tmp/sftp-test
cat > /tmp/sftp-test/order-1.json << 'EOF'
{"order_id": "ORD-001", "customer": "Alice", "amount": 99.99}
EOF

cat > /tmp/sftp-test/order-2.json << 'EOF'
{"order_id": "ORD-002", "customer": "Bob", "amount": 149.50}
EOF

# Upload via SFTP
sftp -o StrictHostKeyChecking=no -P 2222 testuser@localhost << 'EOF'
cd upload
put /tmp/sftp-test/order-1.json
put /tmp/sftp-test/order-2.json
exit
EOF
```

### Step 3: Update Route Configuration

Edit `examples/sftp-ingestion.yaml` and update the `sources` section:

```yaml
sources:
  - name: sftp-source
    type: sftp
    config:
      host: localhost          # Change from sftp.example.com
      port: 2222              # Change from 22
      user: testuser           # testuser for local testing
      password: ${SECRET:SFTP_PASSWORD}
      path: /upload            # Polling directory
      schedule: "30s"          # Poll every 30s for faster testing
```

### Step 4: Run the Example

```bash
# Set test password
export SFTP_PASSWORD="testpass"

# Create output directory
mkdir -p output/sftp-ingested output/sftp-errors

# Run the route
dimctl run examples/sftp-ingestion.yaml
```

### Step 5: Verify File Ingestion

In another terminal, monitor the output:

```bash
# Watch for ingested files
watch -n 1 'ls -lah output/sftp-ingested/ && echo "---" && cat output/sftp-ingested/*.json 2>/dev/null | head -20'

# Or poll once
dimctl run examples/sftp-ingestion.yaml &  # Run in background
sleep 10                                    # Wait for initial poll
ls -la output/sftp-ingested/                # Check output
kill %1                                     # Stop the route
```

Expected output:

```
output/sftp-ingested/order-1.json:
{
  "content": {
    "order_id": "ORD-001",
    "customer": "Alice",
    "amount": 99.99
  },
  "ingested_at": "2026-09-04T12:34:56Z",
  "source_file": "order-1.json",
  "source_host": "localhost",
  "source_path": "/upload/order-1.json"
}
```

## Route Behavior

1. **Polling**: Connects to SFTP server every `schedule` interval (default 5 minutes)
2. **New File Detection**: Tracks file modification times to detect new/changed files
3. **Parsing**: Attempts to parse files as JSON (passes through if not valid JSON)
4. **Enrichment**: Adds metadata including ingestion timestamp and source information
5. **Output**: Successfully processed files go to `processed-files` sink
6. **Error Handling**: Files that fail processing go to `error-sink` (optional)

## Configuration Reference

### SFTP Source Config

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `host` | ✓ | - | SFTP server hostname/IP |
| `port` | | 22 | SFTP server port |
| `user` | ✓ | - | SFTP username |
| `password` | | - | SFTP password (use `${SECRET:VAR_NAME}`) |
| `keyfile` | | - | Path to SSH private key (alternative to password) |
| `path` | ✓ | - | Remote directory to poll |
| `schedule` | | 5m | Poll interval (e.g., "30s", "1m", "5m") |

### Authentication

- **Password**: Set via `password: ${SECRET:SFTP_PASSWORD}` environment variable
- **Private Key**: Set via `keyfile: /path/to/id_rsa` local file path
- **Credentials**: Never hardcode passwords; always use environment variables

## Cleanup

```bash
# Stop SFTP server
docker-compose -f deploy/docker-compose.sftp.yml down

# Remove generated files
rm -rf output/sftp-ingested/ output/sftp-errors/
```

## Exit Criteria (R21.4)

✓ SFTP source configured with password auth connects to real server  
✓ File detection works (new and changed files are detected)  
✓ Files are ingested and written to output  
✓ Demonstrated in runnable example (this document)  
✓ `pollSFTP` no longer returns placeholder error  

## Related

- **R21.1**: SFTP libraries added to go.mod
- **R21.2**: pollSFTP implementation with password/key auth
- **R21.3**: Docker test container and unit tests
- **R21.4**: This end-to-end example
