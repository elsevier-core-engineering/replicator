job "replicator" {
  datacenters = ["us-west-2"]
  region      = "us-west-2"
  type        = "service"

  constraint {
    attribute = "${meta.replicator_enabled}"
    value     = "1"
  }

  update {
    stagger      = "10s"
    max_parallel = 1
  }

  group "nodes" {
    count = 2

    constraint {
      operator  = "distinct_hosts"
      value     = "true"
    }

    constraint {
      attribute = "${meta.enclave}"
      value     = "corp"
    }

    vault {
      change_mode = "noop"
      env = false
      policies = ["dynamic-read"]
    }

    task "replicator" {
      driver = "docker"

      config {
        image        = "583623634344.dkr.ecr.us-west-2.amazonaws.com/replicator:<APP_VERSION>"
        network_mode = "host"
        args         = [
          "agent",
          "-job-scaling-disable",
          "-consul-key-root=replicator/${meta.enclave}/config",
        ]

        volumes = [
          "local/secrets/creds:/root/.aws/credentials"
        ]

        logging {
          type = "syslog"
          config {
            tag = "${NOMAD_JOB_NAME}"
          }
        }
        
      }

      template {
        data = <<EOF
{{ with printf "aws_%s/sts/%s" (env "meta.enclave") (env "NOMAD_JOB_NAME") | secret }}
[default]
aws_access_key_id = {{.Data.access_key}}
aws_secret_access_key = {{.Data.secret_key}}
aws_session_token = {{.Data.security_token}}
{{ end }}
EOF
        destination = "local/secrets/creds"
        change_mode   = "restart"
      }

      resources {
        cpu    = 250
        memory = 60

        network {
          mbits = 2
        }
      }
    }
  }
}
