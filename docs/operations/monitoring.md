# Monitoring Guide

## Overview

Comprehensive monitoring ensures the YouTube Webhook Service operates reliably and efficiently.

## Metrics

### Key Performance Indicators (KPIs)

| Metric | Target | Alert Threshold |
|--------|--------|-----------------|
| Function Error Rate | < 1% | > 5% |
| Response Time (p95) | < 1s | > 3s |
| Subscription Success Rate | > 99% | < 95% |
| Renewal Success Rate | > 98% | < 90% |
| Active Subscriptions | > 0 | = 0 |

### Application Metrics

#### Subscription Metrics
- `subscription.count` - Total active subscriptions
- `subscription.expired` - Expired subscriptions
- `subscription.renewal.success` - Successful renewals
- `subscription.renewal.failure` - Failed renewals
- `subscription.renewal.attempts` - Renewal attempt count

#### Notification Metrics
- `notification.received` - Total notifications received
- `notification.processed` - Successfully processed
- `notification.filtered` - Filtered as non-new videos
- `notification.errors` - Processing errors

#### API Metrics
- `api.requests` - Total API requests
- `api.latency` - Request latency
- `api.errors` - API errors by type
- `github.dispatch.success` - Successful GitHub dispatches
- `github.dispatch.failure` - Failed GitHub dispatches

## Google Cloud Monitoring

### Setup Cloud Monitoring

```bash
# Enable monitoring API
gcloud services enable monitoring.googleapis.com

# Create workspace
gcloud monitoring workspaces create \
  --display-name="YouTube Webhook Monitoring"
```

### Default Metrics

Cloud Functions automatically provides:
- Execution count
- Execution time
- Memory utilization
- Active instances
- Error count

### Custom Metrics

```go
// Example: Track subscription count
import "cloud.google.com/go/monitoring/apiv3"

func recordSubscriptionCount(count int) {
    // Create metric descriptor
    metric := &monitoringpb.TimeSeries{
        Metric: &metricpb.Metric{
            Type: "custom.googleapis.com/subscription/count",
        },
        Points: []*monitoringpb.Point{{
            Value: &monitoringpb.TypedValue{
                Value: &monitoringpb.TypedValue_Int64Value{
                    Int64Value: int64(count),
                },
            },
            Interval: &monitoringpb.TimeInterval{
                EndTime: timestamppb.Now(),
            },
        }},
    }
    
    // Write metric
    client.CreateTimeSeries(ctx, &monitoringpb.CreateTimeSeriesRequest{
        Name:       fmt.Sprintf("projects/%s", projectID),
        TimeSeries: []*monitoringpb.TimeSeries{metric},
    })
}
```

## Logging

### Structured Logging

```go
// Use structured logging for better analysis
log.Printf(
    "subscription_renewal channel_id=%s status=%s attempts=%d error=%v",
    channelID, status, attempts, err,
)
```

### Log Levels

| Level | Usage |
|-------|-------|
| DEBUG | Detailed debugging information |
| INFO | General informational messages |
| WARNING | Warning messages for non-critical issues |
| ERROR | Error messages for failures |
| CRITICAL | Critical failures requiring immediate attention |

### Log Queries

```bash
# View errors
gcloud logging read "resource.type=cloud_function \
  resource.labels.function_name=YouTubeWebhook \
  severity>=ERROR" --limit=50

# View subscription renewals
gcloud logging read "resource.type=cloud_function \
  textPayload:subscription_renewal" --limit=50

# View GitHub dispatch failures
gcloud logging read "resource.type=cloud_function \
  textPayload:github_dispatch AND severity=ERROR" --limit=50
```

## Dashboards

### Create Dashboard

```bash
# Using gcloud
gcloud monitoring dashboards create --config-from-file=dashboard.yaml
```

### Dashboard Configuration

```yaml
displayName: YouTube Webhook Service
gridLayout:
  widgets:
    - title: Request Rate
      xyChart:
        dataSets:
          - timeSeriesQuery:
              timeSeriesFilter:
                filter: metric.type="cloudfunctions.googleapis.com/function/execution_count"
                      resource.type="cloud_function"
                      resource.label.function_name="YouTubeWebhook"
    
    - title: Error Rate
      xyChart:
        dataSets:
          - timeSeriesQuery:
              timeSeriesFilter:
                filter: metric.type="cloudfunctions.googleapis.com/function/error_rate"
                      resource.type="cloud_function"
    
    - title: Active Subscriptions
      scorecard:
        timeSeriesQuery:
          timeSeriesFilter:
            filter: metric.type="custom.googleapis.com/subscription/count"
    
    - title: Memory Usage
      xyChart:
        dataSets:
          - timeSeriesQuery:
              timeSeriesFilter:
                filter: metric.type="cloudfunctions.googleapis.com/function/user_memory_bytes"
```

## Alerting

### Alert Policies

#### High Error Rate Alert

```bash
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="High Error Rate" \
  --condition-display-name="Error rate > 5%" \
  --condition='
    {
      "displayName": "Error rate exceeds 5%",
      "conditionThreshold": {
        "filter": "metric.type=\"cloudfunctions.googleapis.com/function/error_rate\" resource.type=\"cloud_function\"",
        "comparison": "COMPARISON_GT",
        "thresholdValue": 0.05,
        "duration": "60s"
      }
    }'
```

#### Subscription Expiry Alert

```bash
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="No Active Subscriptions" \
  --condition='
    {
      "displayName": "No active subscriptions",
      "conditionThreshold": {
        "filter": "metric.type=\"custom.googleapis.com/subscription/count\"",
        "comparison": "COMPARISON_LT",
        "thresholdValue": 1,
        "duration": "300s"
      }
    }'
```

### Notification Channels

```bash
# Create email channel
gcloud alpha monitoring channels create \
  --display-name="Team Email" \
  --type=email \
  --channel-labels=email_address=team@example.com

# Create Slack channel
gcloud alpha monitoring channels create \
  --display-name="Slack Alerts" \
  --type=slack \
  --channel-labels=channel_name="#alerts"
```

## Health Checks

### Uptime Check

```bash
gcloud monitoring uptime-check-configs create youtube-webhook \
  --display-name="YouTube Webhook Health" \
  --resource-type=UPTIME_URL \
  --monitored-resource="{'type':'uptime_url','labels':{'host':'region-project.cloudfunctions.net','project_id':'PROJECT_ID'}}" \
  --http-check="{'path':'/subscriptions','port':443,'use_ssl':true}" \
  --period=5m
```

### Custom Health Endpoint

```go
// Add health check endpoint
case path == "health" && r.Method == http.MethodGet:
    handleHealthCheck(w, r)

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
    // Check critical dependencies
    checks := map[string]bool{
        "storage": checkStorage(),
        "github": checkGitHub(),
    }
    
    allHealthy := true
    for _, healthy := range checks {
        if !healthy {
            allHealthy = false
            break
        }
    }
    
    response := map[string]interface{}{
        "status": "healthy",
        "checks": checks,
        "timestamp": time.Now(),
    }
    
    if !allHealthy {
        response["status"] = "unhealthy"
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    
    writeJSONResponse(w, response)
}
```

## Performance Monitoring

### Trace Analysis

```go
import "cloud.google.com/go/trace"

func tracedOperation(ctx context.Context) {
    ctx, span := trace.StartSpan(ctx, "operation-name")
    defer span.End()
    
    // Add attributes
    span.AddAttributes(
        trace.StringAttribute("channel_id", channelID),
        trace.Int64Attribute("attempt", attempt),
    )
    
    // Perform operation
    doWork()
}
```

### Latency Tracking

```go
func trackLatency(operation string, start time.Time) {
    duration := time.Since(start)
    log.Printf("METRIC operation=%s duration_ms=%d", operation, duration.Milliseconds())
}

// Usage
start := time.Now()
defer trackLatency("subscription_renewal", start)
```

## Cost Monitoring

### Budget Alerts

```bash
gcloud billing budgets create \
  --billing-account=BILLING_ACCOUNT_ID \
  --display-name="YouTube Webhook Budget" \
  --budget-amount=10 \
  --threshold-rule=percent=50 \
  --threshold-rule=percent=90 \
  --threshold-rule=percent=100
```

### Cost Analysis

```sql
-- BigQuery query for cost analysis
SELECT
  service.description as service,
  sku.description as sku,
  SUM(cost) as total_cost,
  COUNT(*) as usage_count
FROM
  `project.dataset.gcp_billing_export_v1`
WHERE
  service.description = 'Cloud Functions'
  AND usage_start_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY)
GROUP BY
  service, sku
ORDER BY
  total_cost DESC
```

## Incident Response

### Runbook Template

```markdown
## High Error Rate Runbook

### Detection
- Alert: Error rate > 5% for 5 minutes
- Dashboard: YouTube Webhook Service

### Investigation
1. Check recent logs:
   ```bash
   gcloud functions logs read YouTubeWebhook --limit=100 --filter="severity>=ERROR"
   ```
2. Check deployment history:
   ```bash
   gcloud functions versions list YouTubeWebhook
   ```
3. Verify external dependencies (GitHub API, PubSubHubbub)

### Mitigation
1. Rollback if recent deployment:
   ```bash
   gcloud functions deploy YouTubeWebhook --source-version=PREVIOUS_VERSION
   ```
2. Scale up if load-related:
   ```bash
   gcloud functions deploy YouTubeWebhook --max-instances=200
   ```
3. Clear cache if corruption suspected

### Resolution
1. Fix root cause
2. Deploy fix
3. Verify metrics return to normal
4. Update runbook if new issue type
```

## Best Practices

1. **Use Structured Logging**: Makes log analysis easier
2. **Set Reasonable Alerts**: Avoid alert fatigue
3. **Monitor Dependencies**: Track external service health
4. **Regular Reviews**: Review metrics and alerts monthly
5. **Document Incidents**: Create runbooks for common issues
6. **Test Alerts**: Verify alerts work as expected
7. **Cost Tracking**: Monitor costs to avoid surprises

## Phase 4 Implementation (Pending)

Future monitoring enhancements:
- Custom metrics for business KPIs
- Advanced alerting rules
- SLO/SLI tracking
- Distributed tracing
- Performance profiling
- Automated remediation