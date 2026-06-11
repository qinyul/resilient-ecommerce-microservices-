variable "grafana_url" {
  type        = string
  description = "The URL of the Grafana instance"
  default     = "http://localhost:3000"
}

variable "grafana_auth" {
  type        = string
  description = "The credentials or API key for Grafana authentication (username:password)"
  default     = "admin:admin"
  sensitive   = true
}

variable "prometheus_url" {
  type        = string
  description = "The URL of Prometheus from Grafana's perspective inside the Docker network"
  default     = "http://prometheus:9090"
}

variable "loki_url" {
  type        = string
  description = "The URL of Loki from Grafana's perspective inside the Docker network"
  default     = "http://loki:3100"
}

variable "jaeger_url" {
  type        = string
  description = "The URL of Jaeger from Grafana's perspective inside the Docker network"
  default     = "http://jaeger:16686"
}
