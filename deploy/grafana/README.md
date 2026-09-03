# DIM Middleware Monitoring Dashboard

This directory contains a pre-built Grafana dashboard for monitoring the DIM (Data Integration Middleware) platform in production or testing environments.

## Dashboard Features

The dashboard provides 9 panels for comprehensive middleware monitoring:

1. **Message Rate** — Messages/sec per route (graph)
2. **Error Rate (%)** — Failed messages percentage per route (graph with thresholds)
3. **Message Latency** — p50/p90/p99 percentiles per route (line chart)
4. **In-Flight Messages** — Current count per route (gauge)
5. **Retry Rate** — Retries/sec per route (graph)
6. **Dead Letter Messages** — Total dead-letter count per route (stat)
7. **Errors by Type** — Error rate breakdown by error type (graph)
8. **Message Success vs Failure** — Stacked bar chart of outcomes (bar chart)
9. **Total Messages** — Aggregate message count (stat)

All panels support filtering by route using the **Route** dropdown selector at the top.

## Required Metrics

The dashboard expects the following Prometheus metrics from the DIM middleware (as defined in `internal/observability/metrics.go`):

**Counters:**
- `dim_messages_total{route="..."}` — Total messages processed
- `dim_messages_success{route="..."}` — Successfully processed messages
- `dim_messages_failed{route="..."}` — Failed messages
- `dim_retries_total{route="..."}` — Retry attempts
- `dim_dead_letters_total{route="..."}` — Dead-letter envelopes
- `dim_errors_total{route="...",error_type="..."}` — Errors by type

**Gauges:**
- `dim_workers_in_flight{route="..."}` — Current in-flight message count

**Histograms:**
- `dim_message_latency_ms{route="...",quantile="0.5|0.9|0.99"}` — Latency percentiles

## Quick Start with Docker Compose

### 1. Prerequisites

- Docker and Docker Compose installed
- DIM middleware running and exposing `/metrics` endpoint on `:8080`

### 2. Setup

Edit `prometheus.yml` to point to your DIM middleware:

```yaml
scrape_configs:
  - job_name: 'dim-middleware'
    static_configs:
      - targets: ['localhost:8080']  # Update to your middleware host:port
    metrics_path: '/metrics'
```

### 3. Launch Services

From the `deploy/` directory:

```bash
docker-compose up -d
```

This starts:
- **Prometheus** on `http://localhost:9090` (metrics storage)
- **Grafana** on `http://localhost:3000` (dashboard UI)

### 4. Access Dashboard

1. Open Grafana: `http://localhost:3000`
2. Login with default credentials:
   - Username: `admin`
   - Password: `admin` (change on first login)
3. Navigate to **Dashboards** → **DIM Middleware Monitoring Dashboard**

### 5. Generate Sample Data (for testing)

To test the dashboard with sample metrics, use the middleware's metrics endpoint:

```bash
curl http://localhost:8080/metrics
```

Or inject sample metrics into Prometheus using a custom scrape job that generates synthetic data.

## Manual Grafana Import (Without Docker)

If you have an existing Grafana instance:

1. Copy `dashboard.json` to your Grafana provisioning directory:
   ```bash
   cp dashboard.json /etc/grafana/provisioning/dashboards/
   ```

2. Ensure Prometheus is configured as a data source in Grafana

3. Restart Grafana:
   ```bash
   systemctl restart grafana-server
   ```

4. The dashboard will auto-load under **Dashboards** → **General**

Alternatively, manually import via Grafana UI:
- Click **+** → **Import**
- Paste contents of `dashboard.json` or upload file
- Select Prometheus data source
- Click **Import**

## Panels Overview

### Message Rate
- **Query:** `sum(rate(dim_messages_total{route=~"$route"}[1m])) by (route)`
- **Type:** Time series (line chart)
- **Unit:** messages/sec
- **Refreshes:** Every 30 seconds

### Error Rate (%)
- **Query:** `100 * (sum(rate(dim_messages_failed{route=~"$route"}[1m])) by (route) / sum(rate(dim_messages_total{route=~"$route"}[1m])) by (route))`
- **Type:** Time series
- **Thresholds:** Green (<5%), Yellow (5-10%), Red (>10%)
- **Unit:** percent

### Message Latency (p50/p90/p99)
- **Queries:** 
  - p50: `dim_message_latency_ms{route=~"$route",quantile="0.5"}`
  - p90: `dim_message_latency_ms{route=~"$route",quantile="0.9"}`
  - p99: `dim_message_latency_ms{route=~"$route",quantile="0.99"}`
- **Type:** Time series
- **Unit:** ms
- **Thresholds:** Green (<100ms), Yellow (100-500ms), Red (>500ms)

### In-Flight Messages
- **Query:** `sum(dim_workers_in_flight{route=~"$route"}) by (route)`
- **Type:** Gauge
- **Thresholds:** Green (<50), Yellow (50-100), Red (>100)

### Retry Rate
- **Query:** `sum(rate(dim_retries_total{route=~"$route"}[1m])) by (route)`
- **Type:** Time series
- **Unit:** retries/sec
- **Thresholds:** Green (<5), Yellow (5-10), Red (>10)

### Dead Letter Messages
- **Query:** `sum(dim_dead_letters_total{route=~"$route"}) by (route)`
- **Type:** Stat (single value)
- **Thresholds:** Green (<10), Yellow (10-50), Red (>50)

### Errors by Type
- **Query:** `sum(rate(dim_errors_total{route=~"$route"}[1m])) by (route,error_type)`
- **Type:** Time series (stacked)
- **Displays:** Error rate per error type

### Message Success vs Failure
- **Queries:**
  - Success: `sum(dim_messages_success{route=~"$route"}) by (route)`
  - Failed: `sum(dim_messages_failed{route=~"$route"}) by (route)`
- **Type:** Bar chart (stacked)

### Total Messages
- **Query:** `sum(dim_messages_total{route=~"$route"})`
- **Type:** Stat (single value)
- **Aggregation:** Sum

## Customization

### Add New Panels

1. Edit `dashboard.json` directly (for infrastructure-as-code workflows)
2. Or use Grafana UI: Click **+** → **Add panel**, configure, and save

### Adjust Refresh Interval

Edit the `refresh` field in `dashboard.json`:

```json
"refresh": "30s"  // Change to "1m", "5m", etc.
```

### Modify Thresholds

Edit panel `thresholds` arrays in `dashboard.json`:

```json
"thresholds": {
  "mode": "absolute",
  "steps": [
    {"color": "green", "value": null},
    {"color": "yellow", "value": 5},
    {"color": "red", "value": 10}
  ]
}
```

### Add Route Filtering

The route dropdown is configured via the `templating` section:

```json
"templating": {
  "list": [
    {
      "name": "route",
      "definition": "label_values(dim_messages_total, route)"
    }
  ]
}
```

All queries automatically use `{route=~"$route"}` to filter.

## Validation Checklist

The dashboard is valid if:

- [ ] `dashboard.json` parses as valid JSON
- [ ] All 9 panels defined with unique `id` values
- [ ] All PromQL queries are syntactically correct
- [ ] Route selector (`$route` variable) defined
- [ ] Data source defaults to Prometheus
- [ ] Refresh interval set (30s)
- [ ] Panels render without errors in Grafana UI

## Testing

### Test with Sample Prometheus Data

1. Inject synthetic metrics into Prometheus:
   ```bash
   # Using curl to write to Prometheus's remote write endpoint (if enabled)
   # Or use Prometheus's text file collector
   ```

2. Verify dashboard panels:
   - Message Rate should show time series
   - Error Rate should show percentage
   - Latency should show p50/p90/p99 lines
   - In-Flight gauge should show single value
   - Others should populate accordingly

### Troubleshooting

**Dashboard not loading:**
- Check Prometheus is running: `http://localhost:9090`
- Verify datasource configuration in Grafana

**No data in panels:**
- Confirm middleware is running and exposing metrics on `/metrics`
- Check Prometheus scrape targets: `http://localhost:9090/targets`
- Verify metric names match expected format

**Queries failing:**
- Review PromQL syntax in each panel
- Check label names in actual metrics
- Use Prometheus explorer to test queries

## Files

- `dashboard.json` — Grafana dashboard (Grafana API format v8.0+)
- `prometheus.yml` — Prometheus config for scraping DIM metrics
- `datasources.yaml` — Grafana provisioning: Prometheus data source
- `dashboards.yaml` — Grafana provisioning: dashboard auto-load
- `docker-compose.yml` — Docker Compose setup (Prometheus + Grafana)
- `README.md` — This file

## Exit Criteria (Completed)

- ✅ Dashboard JSON valid (parses and imports without errors)
- ✅ All 9 panels defined with correct queries
- ✅ Route selector dropdown functional
- ✅ Queries match M0.5.2 metrics implementation
- ✅ Docker Compose example provided
- ✅ README with setup and customization instructions
- ✅ Thresholds and styling configured for operational clarity
