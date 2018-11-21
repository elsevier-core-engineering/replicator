package replicator

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/d3sw/replicator/client"
	"github.com/d3sw/replicator/logging"
	"github.com/d3sw/replicator/notifier"
	"github.com/d3sw/replicator/replicator/structs"
)

// newJobScalingPolicy returns a new JobScalingPolicies struct for Replicator
// to track and view Nomad job scaling.
func newJobScalingPolicy() *structs.JobScalingPolicies {

	return &structs.JobScalingPolicies{
		Policies: make(map[string][]*structs.GroupScalingPolicy),
		Lock:     sync.RWMutex{},
	}
}

func (s *Server) asyncJobScaling(jobScalingPolicies *structs.JobScalingPolicies) {
	// Setup our wait group to ensure we block until all worker pool scaling
	// operations have completed.
	var wg sync.WaitGroup

	// Get the current number of registered jobs.
	jobCount := len(jobScalingPolicies.Policies)

	// Register an entry to the wait group for each scalable job.
	wg.Add(jobCount)

	// Build a buffered channel to pass our scalable resources to worker threads.
	jobs := make(chan string, jobCount)

	// Calculate the number of worker threads to initiate.
	maxConcurrency := s.config.ScalingConcurrency

	if jobCount < maxConcurrency {
		maxConcurrency = jobCount
	}

	logging.Debug("code/job_scaling: initiating %v concurrent scaling threads "+
		"to process %v jobs", maxConcurrency, jobCount)

	// Initiate workers to implement job scaling.
	for w := 1; w <= maxConcurrency; w++ {
		go s.jobScaling(w, jobs, jobScalingPolicies, &wg)
	}

	// Add jobs to the worker channel.
	for job := range jobScalingPolicies.Policies {
		if s.config.NomadClient.IsJobInDeployment(job) {
			logging.Debug("core/job_scaling: job %s is in deployment, no scaling "+
				"evaluation will be triggered", job)
			continue
		}

		jobs <- job
	}
}

func (s *Server) jobScaling(id int, jobs <-chan string,
	jobScalingPolicies *structs.JobScalingPolicies, wg *sync.WaitGroup) {

	// Setup references to clients for Nomad and Consul.
	nomadClient := s.config.NomadClient
	consulClient := s.config.ConsulClient

	for jobName := range jobs {
		func() {
			defer wg.Done()
			logging.Debug("core/job_scaling: scaling thread %v evaluating scaling "+
				"for job %v", id, jobName)

			g := jobScalingPolicies.Policies[jobName]

			// EvaluateJobScaling performs read/write to our map therefore we wrap it
			// in a read/write lock and remove this as soon as possible as the
			// remaining functions only need a read lock.
			jobScalingPolicies.Lock.Lock()
			err := nomadClient.EvaluateJobScaling(jobName, g)
			jobScalingPolicies.Lock.Unlock()

			// Horrible but required for jobs that have been purged as the policy
			// watcher will not get notified and as such, cannot remove the policy even
			// though the job doesn't exist. The string check is due to
			// github.com/hashicorp/nomad/issues/1849
			if err != nil && strings.Contains(err.Error(), "404") {
				client.RemoveJobScalingPolicy(jobName, jobScalingPolicies)

				return
			} else if err != nil {
				logging.Error("core/job_scaling: unable to perform job resource "+
					"evaluation: %v", err)

				return
			}

			jobScalingPolicies.Lock.RLock()

			for _, group := range g {
				// Setup a failure message to pass to the failsafe check.
				message := &notifier.FailureMessage{
					AlertUID:     group.UID,
					ResourceID:   fmt.Sprintf("%s/%s", jobName, group.GroupName),
					ResourceType: JobType,
				}

				// Read our JobGroup state and check failsafe.
				state := &structs.ScalingState{
					ResourceName: group.GroupName,
					ResourceType: JobType,
					StatePath: s.config.ConsulKeyRoot + "/state/jobs/" + jobName +
						"/" + group.GroupName,
				}
				consulClient.ReadState(state, true)

				if !FailsafeCheck(state, s.config, group.RetryThreshold, message) {
					logging.Error("core/job_scaling: job \"%v\" and group \"%v\" is in "+
						"failsafe mode", jobName, group.GroupName)
					continue
				}

				// Check the JobGroup scaling cooldown.
				cd := time.Duration(group.Cooldown) * time.Second

				if !state.LastScalingEvent.Before(time.Now().Add(-cd)) {
					logging.Debug("core/job_scaling: job \"%v\" and group \"%v\" has not reached scaling cooldown threshold of %s",
						jobName, group.GroupName, cd)
					continue
				}

				if group.ScaleDirection == client.ScalingDirectionOut || group.ScaleDirection == client.ScalingDirectionIn {
					if group.Enabled {
						logging.Debug("core/job_scaling: scaling for job \"%v\" and group \"%v\" is enabled; a "+
							"scaling operation (%v) will be requested", jobName, group.GroupName, group.ScaleDirection)

						// Submit the job and group for scaling.
						nomadClient.JobGroupScale(jobName, group, state)

					} else {
						logging.Debug("core/job_scaling: job scaling has been disabled; a "+
							"scaling operation (%v) would have been requested for \"%v\" "+
							"and group \"%v\"", group.ScaleDirection, jobName, group.GroupName)
					}
				}

				// Persist our state to Consul.
				consulClient.PersistState(state)
			}

			// Release our read-only lock.
			jobScalingPolicies.Lock.RUnlock()
		}()
	}
}
