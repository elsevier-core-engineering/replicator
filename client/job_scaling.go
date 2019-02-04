package client

import (
	"fmt"
	"strings"
	"time"

	metrics "github.com/armon/go-metrics"
	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
	nomad "github.com/hashicorp/nomad/api"
	nomadstructs "github.com/hashicorp/nomad/nomad/structs"
)

const (
	deploymentTimeOut = 15 * time.Minute
	evaluationTimeOut = 30 * time.Second
)

// JobGroupScale scales a particular job group, confirming that the action
// completes successfully.
func (c *nomadClient) JobGroupScale(jobName string, group *structs.GroupScalingPolicy, state *structs.ScalingState) {

	// In order to scale the job, we need information on the current status of the
	// running job from Nomad.
	jobResp, _, err := c.nomad.Jobs().Info(jobName, c.queryOptions())

	if err != nil {
		logging.Error("client/job_scaling: unable to determine job info of %v: %v", jobName, err)
		return
	}

	// Use the current task count in order to determine whether or not a scaling
	// event will violate the min/max job policy.
	for i, taskGroup := range jobResp.TaskGroups {
		if group.GroupName != *taskGroup.Name {
			// There could be multiple groups in a job - make sure we match the group name.
			continue
		}

		if group.ScaleDirection == ScalingDirectionOut && *taskGroup.Count >= group.Max ||
			group.ScaleDirection == ScalingDirectionIn && *taskGroup.Count <= group.Min {
			logging.Debug("client/job_scaling: scale %v not permitted due to constraints on job \"%v\" and group \"%v\", targetCount: %d, GroupMin: %d GroupMax: %d",
				group.ScaleDirection, *jobResp.ID, group.GroupName, *taskGroup.Count, group.Min, group.Max)
			return
		}

		logging.Info("client/job_scaling: scale %v will now be initiated against job \"%v\" and group \"%v\"",
			group.ScaleDirection, jobName, group.GroupName)

		// Depending on the scaling direction decrement/incrament the count;
		// currently replicator only supports addition/subtraction of 1.
		if *taskGroup.Name == group.GroupName && group.ScaleDirection == ScalingDirectionOut {
			*jobResp.TaskGroups[i].Count++
			state.ScaleOutRequests++
		}

		if *taskGroup.Name == group.GroupName && group.ScaleDirection == ScalingDirectionIn {
			*jobResp.TaskGroups[i].Count--
			state.ScaleInRequests++
		}
	}

	// Ensure that job is stable before triggering scale action
	if !*jobResp.Stable {
		logging.Warning("client/job_scaling: Job %s is not stable. Bailing on scale action.", jobName)
		return
	}
	// Submit the job to the Register API endpoint with the altered count number
	// and check that no error is returned.
	resp, _, err := c.nomad.Jobs().Register(jobResp, &nomad.WriteOptions{})

	// Track the scaling submission time.
	state.LastScalingEvent = time.Now()
	if err != nil {
		logging.Error("client/job_scaling: issue submitting job %s for scaling action: %v", jobName, err)
		return
	}

	// Setup our metric scaling direction namespace.
	m := fmt.Sprintf("scale_%s", strings.ToLower(group.ScaleDirection))

	success := c.scaleConfirmation(resp.EvalID)

	if !success {
		metrics.IncrCounter([]string{"job", jobName, group.GroupName, m, "failure"}, 1)
		state.FailureCount++

		return
	}

	metrics.IncrCounter([]string{"job", jobName, group.GroupName, m, "success"}, 1)
	logging.Info("client/job_scaling: scaling of job \"%v\" and group \"%v\" successfully completed",
		jobName, group.GroupName)
}

// scaleConfirmation takes the EvaluationID from the job registration and checks
// via a timer and blocking queries that the resulting deployment completes
// successfully.
func (c *nomadClient) scaleConfirmation(evalID string) (success bool) {
	depID, err := c.getDeploymentID(evalID)
	if err != nil {
		logging.Error("client/job_scaling: unable to obtain evaluation info or "+
			"deployment ID for evaluation %v: %v", evalID, err)
		return
	}

	timeOut := time.After(deploymentTimeOut)
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()

	q := &nomad.QueryOptions{WaitIndex: 1, AllowStale: true}

	for {
		select {
		case <-timeOut:
			logging.Error("client/job_scaling: deployment %s reached timeout %v",
				depID, deploymentTimeOut)
			return

		case <-tick.C:
			dep, meta, err := c.nomad.Deployments().Info(depID, q)
			if err != nil {
				logging.Error("client/job_scaling: unable to list Nomad "+
					"deployment %s: %v", depID, err)
				return
			}

			// Check the LastIndex for an update.
			if meta.LastIndex <= q.WaitIndex {
				continue
			}

			q.WaitIndex = meta.LastIndex

			// Check the deployment status.
			if dep.Status == nomadstructs.DeploymentStatusSuccessful {
				return true
			} else if dep.Status == nomadstructs.DeploymentStatusRunning {
				logging.Debug("client/job_scaling: deployment %s is still in progress", depID)
				continue
			} else {
				return false
			}
		}
	}
}

// getDeploymentID retrieves the deployment ID for a given Nomad evaluation.
func (c *nomadClient) getDeploymentID(evalID string) (depID string, err error) {
	var eval *nomad.Evaluation

	// Setup our retry ticker to keep polling the Nomad API until we get
	// a deployment ID.
	ticker := time.NewTicker(time.Millisecond * 500)
	defer ticker.Stop()

	timeout := time.NewTicker(evaluationTimeOut)
	defer timeout.Stop()

	for {
		select {
		case <-timeout.C:
			return depID, fmt.Errorf("timeout reached while trying to retrieve the "+
				"deployment ID for evaluation %v", evalID)

		case <-ticker.C:
			if eval, _, err = c.nomad.Evaluations().Info(evalID, nil); err != nil {
				logging.Error("client/job_scaling: an error occurred while trying "+
					"to retrieve the deployment ID for evaluation %v: %v", evalID, err)
				continue
			}

			if eval.DeploymentID == "" {
				logging.Debug("client/job_scaling: received an empty deployment for "+
					"evaluation %v; pausing and retrying", evalID)
				continue
			}

			logging.Debug("client/job_scaling: received deployment ID %v for "+
				"evaluation %v", eval.DeploymentID, evalID)
			return eval.DeploymentID, nil
		}
	}
}

// IsJobInDeployment checks to see whether the supplied Nomad job is currently
// in the process of a deployment.
func (c *nomadClient) IsJobInDeployment(jobName string) (isRunning bool) {

	resp, _, err := c.nomad.Jobs().LatestDeployment(jobName, c.queryOptions())

	if err != nil {
		logging.Error("client/job_scaling: unable to list Nomad deployments: %v", err)
		return
	}

	if resp == nil {
		logging.Debug("client/job_scaling: no deployments found for job: %v", jobName)
		return false
	} else {
		switch resp.Status {
		case nomadstructs.DeploymentStatusRunning:
			return true
		case nomadstructs.DeploymentStatusDescriptionPaused:
			return true
		default:
			return false
		}
	}
}
