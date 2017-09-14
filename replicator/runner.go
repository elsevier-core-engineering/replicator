package replicator

import (
	"os"
	"time"

	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
)

// Runner is the main runner struct.
type Runner struct {
	// candaidate is our LeaderCandidate for the runner instance.
	candidate *LeaderCandidate

	// config is the Config that created this Runner. It is used internally to
	// construct other objects and pass data.
	config *structs.Config

	// doneChan is where finish notifications occur.
	doneChan chan bool
}

// NewRunner sets up the Runner type.
func NewRunner(config *structs.Config) (*Runner, error) {
	runner := &Runner{
		doneChan: make(chan bool),
		config:   config,
	}
	return runner, nil
}

// Start is the main entry point into Replicator and launches processes based
// on the configuration.
func (r *Runner) Start() {

	// Setup our LeaderCandidate object for leader elections and session renewal.
	leaderKey := r.config.ConsulKeyRoot + "/" + "leader"
	r.candidate = newLeaderCandidate(r.config.ConsulClient, leaderKey,
		leaderLockTimeout)
	go r.leaderTicker()

	jobScalingPolicy := newJobScalingPolicy()

	if !r.config.ClusterScalingDisable || !r.config.JobScalingDisable {
		// Setup our JobScalingPolicy Watcher and start running this.
		go r.config.NomadClient.JobWatcher(jobScalingPolicy)
	}

	if !r.config.ClusterScalingDisable {
		// Setup the node registry and initiate worker pool and node discovery.
		nodeRegistry := structs.NewNodeRegistry()
		go r.config.NomadClient.NodeWatcher(nodeRegistry)

		// Launch our cluster scaling main ticker function
		go r.clusterScalingTicker(nodeRegistry, jobScalingPolicy)
	}

	// Launch our job scaling main ticker function
	if !r.config.JobScalingDisable {
		go r.jobScalingTicker(jobScalingPolicy)
	}
}

// Stop halts the execution of this runner.
func (r *Runner) Stop() {
	r.candidate.endCampaign()
	r.doneChan <- true
	close(r.doneChan)
}

func (r *Runner) leaderTicker() {
	ticker := time.NewTicker(
		time.Second * time.Duration(leaderElectionInterval),
	)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Perform the leadership locking and continue if we have confirmed that
			// we are running as the replicator leader.
			r.candidate.leaderElection()
		case <-r.doneChan:
			return
		}
	}
}

func (r *Runner) jobScalingTicker(jobPol *structs.JobScalingPolicies) {
	ticker := time.NewTicker(
		time.Second * time.Duration(r.config.JobScalingInterval),
	)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if r.candidate.isLeader() {
				r.asyncJobScaling(jobPol)
			}
		case <-r.doneChan:
			return
		}
	}
}

func (r *Runner) clusterScalingTicker(nodeReg *structs.NodeRegistry, jobPol *structs.JobScalingPolicies) {
	ticker := time.NewTicker(
		time.Second * time.Duration(r.config.ClusterScalingInterval),
	)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if r.candidate.isLeader() {
				// If we're running as a Nomad job, perform a reverse lookup to
				// identify the node on which we're running and register it as
				// protected.
				if allocID := os.Getenv("NOMAD_ALLOC_ID"); len(allocID) > 0 {
					host, err := r.config.NomadClient.NodeReverseLookup(allocID)
					if err != nil {
						logging.Error("core/runner: Running as a Nomad job but unable "+
							"to determine the ID of our host node: %v", err)
					}

					if len(host) > 0 {
						nodeReg.Lock.Lock()
						// Reverse lookup our worker pool name by node ID and register
						// the node as protected.
						if pool, ok := nodeReg.RegisteredNodes[host]; ok {
							logging.Debug("core/runner: registering node %v in worker "+
								"pool %v as protected", host, pool)
							nodeReg.WorkerPools[pool].ProtectedNode = host
						}
						nodeReg.Lock.Unlock()
					}
				}

				r.asyncClusterScaling(nodeReg, jobPol)

			}
		case <-r.doneChan:
			return
		}
	}
}
