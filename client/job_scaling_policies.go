package client

import (
	"time"

	nomad "github.com/hashicorp/nomad/api"
	nomadStructs "github.com/hashicorp/nomad/nomad/structs"
	"github.com/mitchellh/mapstructure"

	"github.com/elsevier-core-engineering/replicator/helper"
	"github.com/elsevier-core-engineering/replicator/logging"
	"github.com/elsevier-core-engineering/replicator/replicator/structs"
)

// JobWatcher is the main entry point into Replicators process of reading and
// updating its JobScalingPolicies tracking.
func (c *nomadClient) JobWatcher(jobScalingPolicies *structs.JobScalingPolicies) {
	q := &nomad.QueryOptions{WaitIndex: 1, AllowStale: true}

	for {
		jobs, meta, err := c.nomad.Jobs().List(q)
		if err != nil {
			logging.Error("client/job_scaling_policies: failed to retrieve jobs from the Nomad API: %v", err)

			// Sleep as we don't want to retry the API call as fast as Go possibly can.
			time.Sleep(20 * time.Second)
			continue
		}

		// If the LastIndex is not greater than our stored LastChangeIndex, we don't
		// need to do anything. On the initial run this will always result in a full
		// run as the LastChangeIndex is initialized to 0.
		if meta.LastIndex <= jobScalingPolicies.LastChangeIndex {
			logging.Debug("client/job_scaling_policies: blocking query timed out, " +
				"restarting job discovery watcher")
			continue
		}

		// Iterate jobs and find events that have changed since last run
		for _, job := range jobs {
			if job.ModifyIndex <= jobScalingPolicies.LastChangeIndex {
				continue
			}

			// Dpending on the status of the job, take different action on the scaling
			// policy struct.
			switch job.Status {
			case nomadStructs.JobStatusRunning:
				go c.jobScalingPolicyProcessor(job.ID, jobScalingPolicies)
			case nomadStructs.JobStatusDead:
				go RemoveJobScalingPolicy(job.ID, jobScalingPolicies)
			default:
				continue
			}
		}

		// Persist the LastIndex into our scaling policy struct.
		jobScalingPolicies.Lock.Lock()
		jobScalingPolicies.LastChangeIndex = meta.LastIndex
		q.WaitIndex = meta.LastIndex
		jobScalingPolicies.Lock.Unlock()
	}
}

// jobScalingPolicyProcessor triggers an iteation of the job groups to determine
// their meta paramerters scaling policy status.
func (c *nomadClient) jobScalingPolicyProcessor(jobID string, scaling *structs.JobScalingPolicies) {

	jobInfo, _, err := c.nomad.Jobs().Info(jobID, c.queryOptions())
	if err != nil {
		logging.Error("client/job_scaling_policies: unable to call Nomad job info: %v", err)
	}

	// If the job type is incompatible with scaling, there is no need to
	// process the configuration.
	if *jobInfo.Type == nomadStructs.JobTypeSystem {
		return
	}

	// It seems when a job is stopped Nomad notifies twice; once indicates the job
	// is in running state, the second time is that the job is dead. This check
	// is to catch that.
	if *jobInfo.Status != nomadStructs.JobStatusRunning {
		return
	}

	// These are our required keys for Replicator
	requiredKeys := []string{
		"replicator_enabled",
		"replicator_min",
		"replicator_max",
		"replicator_scalein_mem",
		"replicator_scalein_cpu",
		"replicator_scaleout_mem",
		"replicator_scaleout_cpu",
		"replicator_notification_uid",
	}

	// Run the checkOrphanedGroup function.
	go checkOrphanedGroup(jobID, jobInfo.TaskGroups, scaling)

	for _, group := range jobInfo.TaskGroups {

		if group.Update == nil {
			logging.Error("client/job_scaling_policies: job %s and group %v is missing update stanza",
				jobID, *group.Name)
			continue
		}

		missedKeys := helper.ParseMetaConfig(group.Meta, requiredKeys)

		// If all 7 keys missed, then the job group does not have scaling enabled,
		// this is logged for operator clarity.
		if len(missedKeys) == len(requiredKeys) {
			logging.Debug("client/job_scaling_policies: job %s and group %v is not configured for autoscaling",
				jobID, *group.Name)
			go removeGroupScalingPolicy(jobID, *group.Name, scaling)
			continue
		}

		// If some keys missed, the operator has made an effort to enable job scaling
		// but potentially made a typo. This is logged as an error so operators can
		// see and quickly resolve these issues.
		if len(missedKeys) > 0 && len(missedKeys) < len(requiredKeys) {
			logging.Error("client/job_scaling_policies: job %s and group %v is missing meta scaling key(s): %v",
				jobID, *group.Name, missedKeys)
			continue
		}

		// If all keys were matched we update the job scaling policy struct with the
		// information.
		if len(missedKeys) == 0 {
			logging.Debug("client/job_scaling_policies: job %s and group %v has all meta required for autoscaling",
				jobID, *group.Name)

			// Create copies since this runs in a co-routine so that scope is locked in.
			groupName := *group.Name
			groupMeta := make(map[string]string)
			
			for k,v := range group.Meta {
				groupMeta[k] = v
			}
			
			go func() {
				err := updateScalingPolicy(jobID, groupName, groupMeta, scaling)
				if err != nil {
					logging.Error("client/job_scaling_policies: unable to update scaling policy for job %v and group %v: %v",
						jobID, group.Name, err)
				}
			}()
		}
	}
}

// updateScalingPolicy takes a JobGroups meta parameter and updates Replicators
// JobScaling entry if required.
func updateScalingPolicy(jobName, groupName string, groupMeta map[string]string,
	s *structs.JobScalingPolicies) (err error) {

	result := structs.NewGroupScalingPolicy()
	found := false

	// Make use of mapstructures WeaklyTypedInput and setup the decoder. We use
	// WeaklyTypedInput as all meta KVs are strings, whereas we need most of them
	// to differ from this.
	decodeConf := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           result,
	}
	decoder, err := mapstructure.NewDecoder(decodeConf)
	if err != nil {
		return
	}

	// Decode the meta and add the group name to the correct field as this is not
	// available in the meta.
	if err = decoder.Decode(groupMeta); err != nil {
		return
	}

	result.GroupName = groupName
	s.Lock.Lock()

	// If the job already has an entry in the scaling policies, attempt to find
	// the group and overwrite with the new policy. If the job is found, but no
	// group policy is found we append the new group policy to the job.
	if val, ok := s.Policies[jobName]; ok {
		for i, group := range val {
			if group.GroupName == groupName {
				found = true
				changed, err := helper.HasObjectChanged(result, val[i])
				if err != nil {
					logging.Error("client/job_scaling_policies: unable to determine if the job group policy has been updated: %v", err)
				}

				if !changed {
					continue
				}

				val[i] = result
				logging.Info("client/job_scaling_policies: updated scaling policy for job %s and group %s",
					jobName, groupName)

			} else {
				s.Policies[jobName] = append(s.Policies[jobName], result)
				logging.Info("client/job_scaling_policies: added new scaling policy for job %s and group %s",
					jobName, groupName)
				found = true
			}
		}
	}

	// If the job and group have not been found, create a new entry for the job
	// and add the group policy.
	if !found {
		s.Policies[jobName] = append(s.Policies[jobName], result)
		logging.Info("client/job_scaling_policies: added new policy for job %s and group %s",
			jobName, groupName)
	}
	s.Lock.Unlock()
	return
}

// removeScalingPolicy will remove a particular JobGroups scaling policy and
// will also remove the Job entry from the map if there are no longer any Group
// policies associated to it. This is used for jobs which are still running.
func removeGroupScalingPolicy(jobName, groupName string, scaling *structs.JobScalingPolicies) {
	if val, ok := scaling.Policies[jobName]; ok {
		for i, group := range val {
			if group.GroupName == groupName {
				scaling.Lock.Lock()
				scaling.Policies[jobName] = append(scaling.Policies[jobName][:i], scaling.Policies[jobName][i+1:]...)
				scaling.Lock.Unlock()
				logging.Info("client/job_scaling_policies: removed policy for job %s and group %s",
					jobName, groupName)
			}
		}
		if len(scaling.Policies[jobName]) == 0 {
			scaling.Lock.Lock()
			delete(scaling.Policies, jobName)
			scaling.Lock.Unlock()
		}
	}
}

// RemoveJobScalingPolicy deletes the job entry within the the policies map.
func RemoveJobScalingPolicy(jobName string, scaling *structs.JobScalingPolicies) {
	if _, ok := scaling.Policies[jobName]; ok {
		scaling.Lock.Lock()
		delete(scaling.Policies, jobName)
		scaling.Lock.Unlock()
		logging.Info("client/job_scaling_policies: deleted job scaling entries for job %v", jobName)
	}
}

// checkOrphanedGroup checks whether a job has been updated and removed a group
// which has a scaling policy; thus removing the entry.
func checkOrphanedGroup(jobName string, groups []*nomad.TaskGroup, scaling *structs.JobScalingPolicies) {

	taskGroupNames := make([]string, 0)
	taskGroupPolicyNames := make([]string, 0)

	scaling.Lock.RLock()
	if val, ok := scaling.Policies[jobName]; ok {
		for _, g := range val {
			taskGroupPolicyNames = append(taskGroupPolicyNames, g.GroupName)
		}
	}

	for _, g := range groups {
		taskGroupNames = append(taskGroupNames, *g.Name)
	}

	for _, g := range taskGroupNames {
		for i, gp := range taskGroupPolicyNames {
			if g == gp {
				taskGroupPolicyNames = append(taskGroupPolicyNames[:i], taskGroupPolicyNames[i+1:]...)
			}
		}
	}
	scaling.Lock.RUnlock()

	for _, g := range taskGroupPolicyNames {
		removeGroupScalingPolicy(jobName, g, scaling)
	}
}
