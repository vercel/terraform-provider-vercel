resource "vercel_project" "example" {
  name = "firewall-config-example"
}

resource "vercel_firewall_config" "example" {
  project_id = vercel_project.example.id

  rules {
    rule {
      name        = "Bypass Known request"
      description = "Bypass requests using internal bearer tokens"
      # individual condition groups are evaluated as ORs
      condition_group = [
        {
          conditions = [{
            type  = "header"
            key   = "Authorization"
            op    = "eq"
            value = "Bearer internaltoken"
          }]
        },
        {
          conditions = [{
            type  = "header"
            key   = "Authorization"
            op    = "eq"
            value = "Bearer internaltoken2"
          }]
        }
      ]
      action = {
        action = "bypass"
      }
    }

    rule {
      name        = "Challenge curl"
      description = "Challenge user agents containing 'curl'"
      condition_group = [{
        conditions = [{
          type  = "user_agent"
          op    = "sub"
          value = "curl"
        }]
      }]
      action = {
        action = "challenge"
      }
    }

    rule {
      name        = "Deny cookieless requests"
      description = "requests to /api that are missing a session cookie"
      # multiple conditions in a single condition group are evaluated as ANDs
      condition_group = [{
        conditions = [{
          type  = "path"
          op    = "eq"
          value = "/api"
          },
          {
            # Use 'nex' (not exists) to check if cookie is missing
            # Note: no 'value' field needed for existence operators (ex/nex)
            type = "cookie"
            key  = "_session"
            neg  = true
            op   = "ex"
        }]
      }]
      action = {
        action = "challenge"
      }
    }

    rule {
      name        = "Require Authorization header"
      description = "Block requests without Authorization header"
      condition_group = [{
        conditions = [{
          # 'nex' (not exists) checks if header is absent
          type = "header"
          key  = "Authorization"
          op   = "nex"
        }]
      }]
      action = {
        action = "deny"
      }
    }

    rule {
      name        = "Log requests with custom header"
      description = "Log requests that have X-Custom-Header present"
      condition_group = [{
        conditions = [{
          # 'ex' (exists) checks if header is present
          type = "header"
          key  = "X-Custom-Header"
          op   = "ex"
        }]
      }]
      action = {
        action = "log"
      }
    }

    rule {
      name        = "Rate limit API"
      description = "apply ratelimit to requests under /api"
      condition_group = [{
        conditions = [{
          type  = "path"
          op    = "pre"
          value = "/api"
        }]
      }]

      action = {
        action = "rate_limit"
        rate_limit = {
          limit  = 100
          window = 300
          keys   = ["ip", "ja4"]
          algo   = "fixed_window"
          action = "deny"
        }
        action_duration = "5m"
      }
    }

    rule {
      name        = "Known clients"
      description = "Match known keys in header"
      condition_group = [{
        conditions = [{
          type = "header"
          key  = "Authorization"
          op   = "inc"
          values = [
            "key1",
            "key2",
          ]
        }]
      }]

      action = {
        action = "rate_limit"
        rate_limit = {
          limit  = 100
          window = 300
          keys   = ["ip", "ja4"]
          algo   = "fixed_window"
          action = "deny"
        }
        action_duration = "5m"
      }
    }
  }
}

resource "vercel_project" "managed_example" {
  name = "firewall-managed-rule-example"
}

resource "vercel_firewall_config" "managed" {
  project_id = vercel_project.managed.id

  managed_rulesets {
    owasp {
      xss  = { action = "deny" }
      sqli = { action = "deny" }
      rce  = { action = "deny" }
      php  = { action = "deny" }
      java = { action = "deny" }
      lfi  = { action = "deny" }
      rfi  = { action = "deny" }
      gen  = { action = "deny" }
    }

    bot_protection {
      action = "log"
      active = true
    }

    ai_bots {
      action = "log"
      active = true
    }
  }
}

resource "vercel_project" "ip_example" {
  name = "firewall-ip-blocking-example"
}

resource "vercel_firewall_config" "ip-blocking" {
  project_id = vercel_project.ip_example.id

  ip_rules {
    # deny this subnet for all my hosts
    rule {
      action   = "deny"
      ip       = "51.85.0.0/16"
      hostname = "*"
    }

    rule {
      action   = "challenge"
      ip       = "1.2.3.4"
      hostname = "example.com"
    }
  }
}
