job "replicator" {
  datacenters = ["dc1"]
  region      = "global"
  type        = "service"

  update {
    stagger      = "10s"
    max_parallel = 1
  }

  meta {
    VERSION = "v0.0.2"
  }

  group "replicator" {
    count = 2

    task "replicator" {
      driver = "exec"

      constraint {
        attribute = "${attr.kernel.name}"
        value     = "linux"
      }

      config {
        command = "${attr.kernel.name}-${attr.cpu.arch}-replicator"
        args    = [
          "agent",
          "-aws-region=us-east-1",
          "-consul-token=CONSUL_ACL_TOKEN",
          "-cluster-autoscaling-group=WORKER_POOL_ASG_NAME",
        ]
      }

      artifact {
        source = "https://github.com/elsevier-core-engineering/replicator/releases/download/${NOMAD_META_VERSION}/${attr.kernel.name}-${attr.cpu.arch}-replicator"
      }

      resources {
        cpu    = 250
        memory = 60

        network {
          mbits = 5
        }
      }
    }
  }
}
