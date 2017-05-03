package replicator

import (
	"time"

	"github.com/elsevier-core-engineering/replicator/api"
	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
)

// Runner is the main runner struct.
type Runner struct {
	// doneChan is where finish notifications occur.
	doneChan chan struct{}

	// config is the Config that created this Runner. It is used internally to
	// construct other objects and pass data.
	config *structs.Config
}

// NewRunner sets up the Runner type.
func NewRunner(config *structs.Config) (*Runner, error) {
	runner := &Runner{
		doneChan: make(chan struct{}),
		config:   config,
	}
	return runner, nil
}

// Start creates a new runner and uses a ticker to block until the doneChan is
// closed at which point the ticker is stopped.
func (r *Runner) Start() {
	ticker := time.NewTicker(time.Second * time.Duration(10))

	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:

			clusterChan := make(chan bool)
			go r.clusterScaling(clusterChan)
			<-clusterChan

			r.jobScaling()

		case <-r.doneChan:
			return
		}
	}
}

// Stop halts the execution of this runner.
func (r *Runner) Stop() {
	close(r.doneChan)
}

// clusterScaling is the main entry point into the cluster scaling functionality
// and ties numerous functions together to create an asynchronus function which
// can be called from the runner.
func (r *Runner) clusterScaling(done chan bool) {
	nomadClient := r.config.NomadClient
	scalingEnabled := r.config.ClusterScaling.Enabled

	// Determine if we are running on the leader node, halt if not.
	if haveLeadership := nomadClient.LeaderCheck(); !haveLeadership {
		logging.Debug("core/runner: replicator is not running on the known leader, no cluster scaling actions will be taken")
		done <- true
		return
	}

	// If a region has not been specified, attempt to dynamically determine what
	// region we are running in.
	if r.config.Region == "" {
		if region, err := api.DescribeAWSRegion(); err == nil {
			r.config.Region = region
		}
	}

	clusterCapacity := &structs.ClusterAllocation{}

	if scale, err := nomadClient.EvaluateClusterCapacity(clusterCapacity, r.config); err != nil || !scale {
		logging.Debug("core/runner: scaling operation not required or permitted")
	} else {
		// If we reached this point we will be performing AWS interaction so we
		// create an client connection.
		asgSess := api.NewAWSAsgService(r.config.Region)

		if clusterCapacity.ScalingDirection == api.ScalingDirectionOut {
			if !scalingEnabled {
				logging.Debug("core/runner: cluster scaling disabled, not initiating scaling operation (scale-out)")
				done <- true
				return
			}

			if err := api.ScaleOutCluster(r.config.ClusterScaling.AutoscalingGroup, asgSess); err != nil {
				logging.Error("core/runner: unable to successfully scale out cluster: %v", err)
			}
		}

		if clusterCapacity.ScalingDirection == api.ScalingDirectionIn {
			nodeID, nodeIP := nomadClient.LeastAllocatedNode(clusterCapacity)
			if nodeIP != "" && nodeID != "" {
				if !scalingEnabled {
					logging.Debug("core/runner: cluster scaling disabled, not initiating scaling operation (scale-in)")
					done <- true
					return
				}

				if err := nomadClient.DrainNode(nodeID); err == nil {
					logging.Info("core/runner: terminating AWS instance %v", nodeIP)
					err := api.ScaleInCluster(r.config.ClusterScaling.AutoscalingGroup, nodeIP, asgSess)
					if err != nil {
						logging.Error("core/runner: unable to successfully terminate AWS instance %v: %v", nodeID, err)
					}
				}
			}
		}
	}
	done <- true
	return
}

// jobScaling is the main entry point for the Nomad job scaling functionality
// and ties together a number of functions to be called from the runner.
func (r *Runner) jobScaling() {

	// Scaling a Cluster Jobs requires access to both Consul and Nomad therefore
	// we setup the clients here.
	consulClient := r.config.ConsulClient

	nomadClient := r.config.NomadClient

	// Determine if we are running on the leader node, halt if not.
	if haveLeadership := nomadClient.LeaderCheck(); !haveLeadership {
		logging.Debug("core/runner: replicator is not running on the known leader, no job scaling actions will be taken")
		return
	}

	// Pull the list of all currently running jobs which have a defined scaling
	// document. Fail quickly if we can't retrieve this list.
	resp, err := consulClient.GetJobScalingPolicies(r.config, nomadClient)
	if err != nil {
		logging.Error("core/runner: failed to determine if any jobs have scaling policies enabled \n%v", err)
		return
	}

	// EvaluateJobScaling identifies whether each of the Job.Groups requires a
	// scaling event to be triggered. This is then iterated so the individual
	// groups can be assesed.
	nomadClient.EvaluateJobScaling(resp)
	for _, job := range resp {

		// Due to the nested nature of the job and group Nomad definitions a dumb
		// metric is used to determine whether the job has 1 or more groups which
		// require scaling.
		i := 0

		for _, group := range job.GroupScalingPolicies {
			if group.Scaling.ScaleDirection == api.ScalingDirectionOut || group.Scaling.ScaleDirection == api.ScalingDirectionIn {
				logging.Debug("core/runner: scaling operation (%v) to be requested on job \"%v\" and group \"%v\"",
					group.Scaling.ScaleDirection, job.JobName, group.GroupName)
				i++
			}
		}

		// If 1 or more groups need to be scaled we submit the whole job for scaling
		// as to scale you must submit the whole job file currently. The JobScale
		// function takes care of scaling groups independently.
		if i > 0 {
			if !job.Enabled {
				logging.Debug("core/runner: scaling disabled for job \"%v\"; no scaling operation will be initiated",
					job.JobName)
				continue
			}

			nomadClient.JobScale(job)
		}
	}
}
