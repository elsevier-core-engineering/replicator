# Replicator

Replicator is a fast and highly concurrent Go daemon that provides dynamic scaling of [Nomad](https://github.com/hashicorp/nomad) jobs and worker nodes.

- Replicator job scaling policies are configured as [meta parameters](https://www.nomadproject.io/docs/job-specification/meta.html) within the job specification. A job scaling policy allows scaling constraints to be defined per task-group. Currently supported scaling metrics are CPU and Memory; there are plans for additional metrics as well as different metric backends in the future. Details of configuring job scaling and other important information can be found on the Replicator [Job Scaling wiki page](https://github.com/d3sw/replicator/wiki/Job-Scaling).

- Replicator supports dynamic scaling of multiple, distinct cluster worker nodes in an AWS autoscaling group. Worker pool autoscaling is configured through Nomad client [meta parameters](https://www.nomadproject.io/docs/agent/configuration/client.html#meta). Details of configuring worker pool scaling and other important information can be found on the Replicator [Cluster Scaling wiki page](https://github.com/d3sw/replicator/wiki/Cluster-Scaling).

*At present, worker pool autoscaling is only supported on AWS, however, future support for GCE and Azure are planned using the Go factory/provider pattern.*

### Download

Pre-compiled releases for a number of platforms are available on the [GitHub release page](https://github.com/d3sw/replicator/releases).

## Running

Replicator can be run in a number of ways; the recommended way is as a Nomad service job either using the [Docker driver](https://www.nomadproject.io/docs/drivers/docker.html) or the [exec driver](https://www.nomadproject.io/docs/drivers/exec.html). There are example Nomad [job specification files](https://github.com/d3sw/replicator/tree/master/example-jobs) available as a starting point.

It's recommended to take a look at the agent [configuration options](https://github.com/d3sw/replicator/wiki/Agent-Command) to configure Replicator to run best in your environment.

Replicator is fully capable of running as a distributed service; using [Consul sessions](https://www.consul.io/docs/internals/sessions.html) to provide leadership locking and exclusion. State is also written by Replicator to the Consul KV store, allowing Replicator failures to be handled quickly and efficiently.

An example Nomad client configuration that can be used to enable autoscaling on the worker pool:
```hcl
bind_addr = "0.0.0.0"
client {
  enabled =  true
  meta {
    "replicator_cooldown"            = 400
    "replicator_enabled"              = true
    "replicator_node_fault_tolerance" = 1
    "replicator_notification_uid"     = "REP2"
    "replicator_provider"             = "aws"
    "replicator_region"               = "us-east-1"
    "replicator_retry_threshold"      = 3
    "replicator_scaling_threshold"    = 3
    "replicator_scale_factor"         = 1
    "replicator_worker_pool"          = "container-node-public-prod"
  }
}
```

An example job which has autoscaling enabled:
```hcl
job "example" {
  datacenters = ["dc1"]
  type        = "service"

  update {
    max_parallel = 1
    stagger      = "10s"
  }

  group "cache" {
    count = 3

    meta {
      "replicator_max"               = 10
      "replicator_cooldown"          = 50
      "replicator_enabled"           = true
      "replicator_min"               = 1
      "replicator_retry_threshold"   = 1
      "replicator_scalein_mem"       = 30
      "replicator_scalein_cpu"       = 30
      "replicator_scaleout_mem"      = 80
      "replicator_scaleout_cpu"      = 80
      "replicator_notification_uid"  = "REP1"
    }

    task "redis" {
      driver = "docker"
      config {
        image = "redis:3.2"
        port_map {
          db = 6379
        }
      }

      resources {
        cpu    = 500 # 500 MHz
        memory = 256 # 256MB
        network {
          mbits = 10
          port "db" {}
        }
      }

      service {
        name = "global-redis-check"
        tags = ["global", "cache"]
        port = "db"
        check {
          name     = "alive"
          type     = "tcp"
          interval = "10s"
          timeout  = "2s"
        }
      }
    }
  }
}
```

### Permissions

Replicator requires permissions to Consul and the AWS (the only currently supported cloud provider) API in order to function correctly. The Consul ACL token is passed as a configuration parameter and AWS API access should be granted using an EC2 instance IAM role. Vault support is planned for the near future, which will change the way in which permissions are managed and provide a much more secure method of delivering these.

#### Consul ACL Token Permissions

If the Consul cluster being used is running ACLs; the following ACL policy will allow Replicator the required access to perform all functions based on its default configuration:

```hcl
key "" {
  policy = "read"
}
key "replicator/config" {
  policy = "write"
}
node "" {
  policy = "read"
}
node "" {
  policy = "write"
}
session "" {
  policy = "read"
}
session "" {
  policy = "write"
}
```

#### AWS IAM Permissions

Until Vault integration is added, the instance pool which is capable of running the Replicator daemon requires the following IAM permissions in order to perform worker pool scaling:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "AuthorizeAutoScalingActions",
            "Action": [
                "autoscaling:DescribeAutoScalingGroups",
                "autoscaling:DescribeAutoScalingInstances",
                "autoscaling:DescribeScalingActivities",
                "autoscaling:DetachInstances",
                "autoscaling:UpdateAutoScalingGroup"
            ],
            "Effect": "Allow",
            "Resource": "*"
        },
        {
            "Sid": "AuthorizeEC2Actions",
            "Action": [
                "ec2:DescribeInstances",
                "ec2:DescribeRegions",
                "ec2:TerminateInstances",
                "ec2:DescribeInstanceStatus"
            ],
            "Effect": "Allow",
            "Resource": "*"
        }
    ]
}
```

### Commands

Replicator supports a number of commands (CLI) which allow for the easy control and manipulation of the replicator binary. In-depth documentation about each command can be found on the Replicator [commands wiki page](https://github.com/d3sw/replicator/wiki/Commands).

#### Command: `agent`

The `agent` command is the main entry point into Replicator. A subset of the available replicator agent configuration can optionally be passed in via CLI arguments and the configuration parameters passed via CLI flags will always take precedent over parameters specified in configuration files.

Detailed information regarding the available CLI flags can be found in the Replicator [agent command wiki page](https://github.com/d3sw/replicator/wiki/Agent-Command).

#### Command: `failsafe`

The `failsafe` command is used to toggle failsafe mode across the pool of Replicator agents. Failsafe mode prevents any Replicator agent from taking any scaling actions on the resource placed into failsafe mode.

Detailed information about failsafe mode operations and the available CLI options can be found in the Replicator [failsafe command wiki page](https://github.com/d3sw/replicator/wiki/Failsafe-Command).

#### Command: `init`

The `init` command creates example job scaling and worker pool scaling meta documents in the current directory. These files provide a starting example for configuring both scaling functionalities.

#### Command: `version`

The `version` command displays build information about the running binary, including the release version.

## Frequently Asked Questions

### When does Replicator adjust the size of the worker pool?

Replicator will dynamically scale-in the worker pool when:
- Resource utilization falls below the capacity required to run all current jobs while sustaining the configured node fault-tolerance. When calculating required capacity, Replicator includes scaling overhead required to increase the count of all running jobs by one.
- Before removing a worker node, Replicator simulates capacity thresholds if we were to remove a node. If the new required capacity is within 10% of the current utilization, Replicator will decline to remove a node to prevent thrashing.

Replicator will dynamically scale-out the worker pool when:
- Resource utilization exceeds or closely approaches the capacity required to run all current jobs while sustaining the configured node fault-tolerance. When calculating required capacity, Replicator includes scaling overhead required to increase the count of all running jobs by one.

### When does Replicator perform scaling actions against running jobs?

Replicator will dynamically scale a job when:
- A valid scaling policy for the job task-group is present within the job specification meta parameters and has the enabled flag set to true.
- A job specification can consist of multiple groups, each group can contain multiple tasks. Resource allocations and count are specified at the group level.
- Replicator evaluates scaling thresholds against the resource requirements defined within a group task. If any task within a group is found to violate the scaling thresholds, the group count will be adjusted accordingly.

## Contributing

Contributions to Replicator are very welcome! Please refer to our [contribution guide](https://github.com/d3sw/replicator/blob/master/.github/CONTRIBUTING.md) for details about hacking on Replicator.

