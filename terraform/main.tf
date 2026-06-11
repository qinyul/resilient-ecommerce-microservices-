# 1. Traces (Must be defined first so others can reference its UID)
resource "grafana_data_source" "jaeger" {
  type = "jaeger"
  name = "Jaeger"
  url  = var.jaeger_url
}

# 2. Metrics (With Exemplars linking to Jaeger)
resource "grafana_data_source" "prometheus" {
  type  = "prometheus"
  name  = "Prometheus"
  url   = var.prometheus_url

  json_data_encoded = jsonencode({
    httpMethod    = "POST"
    timeInterval  = "15s"

    # Enable Exemplars: Clicking a spike in a Prometheus graph opens the trace
    exemplarTraceIdDestinations = [
      {
        datasourceUid = grafana_data_source.jaeger.uid
        name          = "trace_id"
      }
    ]
  })
}

# 3. Logs (With derived fields linking to Jaeger)
resource "grafana_data_source" "loki" {
  type  = "loki"
  name  = "Loki"
  url   = var.loki_url

  json_data_encoded = jsonencode({
    maxLines  = 1000

    # The regex that scans logs for trace IDs and builds the Jaeger URL
    derivedFields = [
      {
        datasourceUid = grafana_data_source.jaeger.uid
        matcherRegex  = "\"trace_id\":\"(\\w+)\""
        name          = "TraceID"
        url           = "$${__value.raw}"
      }
    ]
  })
}

# 4. The Dashboard
resource "grafana_dashboard" "ecommerce" {
  config_json = templatefile("${path.module}/dashboards/ecommerce_dashboard.json", {
    prometheus_uid = grafana_data_source.prometheus.uid
    loki_uid = grafana_data_source.loki.uid
    jaeger_uid = grafana_data_source.jaeger.uid
  })
  overwrite = true
}
