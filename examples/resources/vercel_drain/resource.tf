resource "vercel_project" "example" {
  name = "example-project"
}

# Basic HTTP drain for logs
resource "vercel_drain" "basic_http" {
  name     = "basic-http-logs"
  projects = "all"

  schemas = {
    log = {
      version = "v1"
    }
  }

  delivery = {
    type = "http"
    endpoint = {
      url = "https://example.com/webhook"
    }
    encoding = "json"
    headers = {
      "Authorization" = "Bearer your-token"
    }
  }
}

# Advanced drain with multiple schemas and sampling
resource "vercel_drain" "advanced" {
  name        = "advanced-multi-schema"
  projects    = "some"
  project_ids = [vercel_project.example.id]
  filter      = "level >= 'info'"

  schemas = {
    log = {
      version = "v1"
    }
    trace = {
      version = "v1"
    }
    analytics = {
      version = "v1"
    }
    speed_insights = {
      version = "v1"
    }
  }

  delivery = {
    type = "http"
    endpoint = {
      url = "https://example.com/advanced-drain"
    }
    encoding    = "ndjson"
    compression = "gzip"
    headers = {
      "Authorization" = "Bearer advanced-token"
      "Content-Type"  = "application/x-ndjson"
      "X-Custom"      = "custom-header"
    }
    secret = "your-signing-secret-for-verification"
  }

  sampling = [
    {
      type        = "head_sampling"
      rate        = 0.1
      environment = "production"
    },
    {
      type         = "head_sampling"
      rate         = 0.5
      environment  = "preview"
      request_path = "/api/"
    }
  ]

  transforms = [
    {
      id = "transform-filter-pii"
    },
    {
      id = "transform-enrich-context"
    }
  ]
}

# OTLP HTTP drain for traces
resource "vercel_drain" "otlp_traces" {
  name     = "jaeger-traces"
  projects = "all"

  schemas = {
    trace = {
      version = "v1"
    }
  }

  delivery = {
    type = "otlphttp"
    endpoint = {
      traces = "https://jaeger.example.com/api/traces"
    }
    encoding = "proto"
    headers = {
      "Authorization" = "Bearer jaeger-token"
    }
  }

  sampling = [
    {
      type = "head_sampling"
      rate = 0.01 # 1% sampling for traces
    }
  ]
}
