package client

import (
	"reflect"
	"testing"

	"github.com/d3sw/replicator/replicator/structs"
	nomad "github.com/hashicorp/nomad/api"
)

func TestJobScalingPolicies_updateScalingPolicy(t *testing.T) {
	scaling := exampleJobScalingPolicies()

	jobName1 := "example"
	groupName1 := "cache"
	jobName2 := "woz"
	groupName2 := "jobs"
	groupName3 := "hertzfeld"

	metaKeys := make(map[string]string)
	metaKeys["replicator_enabled"] = "true"
	metaKeys["replicator_max"] = "10000"
	metaKeys["replicator_min"] = "7500"
	metaKeys["replicator_scalein_mem"] = "40"
	metaKeys["replicator_scalein_cpu"] = "40"
	metaKeys["replicator_scaleout_mem"] = "90"
	metaKeys["replicator_scaleout_cpu"] = "90"
	metaKeys["replicator_notification_uid"] = "ELS2"
	metaKeys["replicator_retry_threshold"] = "10"

	updateScalingPolicy(jobName1, groupName1, metaKeys, scaling)
	updateScalingPolicy(jobName2, groupName2, metaKeys, scaling)
	updateScalingPolicy(jobName2, groupName3, metaKeys, scaling)

	expected := &structs.JobScalingPolicies{
		Policies: make(map[string][]*structs.GroupScalingPolicy),
	}
	policy1 := &structs.GroupScalingPolicy{
		GroupName:      "cache",
		Cooldown:       60,
		Enabled:        true,
		Min:            7500,
		Max:            10000,
		ScaleInMem:     40,
		ScaleInCPU:     40,
		ScaleOutMem:    90,
		ScaleOutCPU:    90,
		RetryThreshold: 10,
		UID:            "ELS2",
	}
	policy2 := &structs.GroupScalingPolicy{
		GroupName:      "jobs",
		Cooldown:       60,
		Enabled:        true,
		Min:            7500,
		Max:            10000,
		ScaleInMem:     40,
		ScaleInCPU:     40,
		ScaleOutMem:    90,
		ScaleOutCPU:    90,
		RetryThreshold: 10,
		UID:            "ELS2",
	}
	policy3 := &structs.GroupScalingPolicy{
		GroupName:      "hertzfeld",
		Cooldown:       60,
		Enabled:        true,
		Min:            7500,
		Max:            10000,
		ScaleInMem:     40,
		ScaleInCPU:     40,
		ScaleOutMem:    90,
		ScaleOutCPU:    90,
		RetryThreshold: 10,
		UID:            "ELS2",
	}
	expected.Policies["example"] = append(expected.Policies["example"], policy1)
	expected.Policies["woz"] = append(expected.Policies["woz"], policy2)
	expected.Policies["woz"] = append(expected.Policies["woz"], policy3)

	if !reflect.DeepEqual(scaling.Policies, expected.Policies) {
		t.Fatalf("expected \n%#v\n\n, got \n\n%#v\n\n", expected.Policies, scaling.Policies)
	}
}

func TestJobScalingPolicies_removeScalingPolicy(t *testing.T) {
	scaling := exampleJobScalingPolicies()
	removeGroupScalingPolicy("example", "cache", scaling)

	if len(scaling.Policies) != 0 {
		t.Fatalf("expected empty map return, got %v entries", len(scaling.Policies))
	}
}

func TestJobScalingPolicies_removeJobScalingPolicy(t *testing.T) {
	scaling := exampleJobScalingPolicies()
	RemoveJobScalingPolicy("example", scaling)

	if len(scaling.Policies) != 0 {
		t.Fatalf("expected empty map return, got %v entries", len(scaling.Policies))
	}
}

func TestJobScalingPolicies_checkOrphanedGroup(t *testing.T) {
	scaling := exampleJobScalingPolicies()
	expected := exampleJobScalingPolicies()
	groupName1 := "cache"

	groups := []*nomad.TaskGroup{}
	taskGtoup := &nomad.TaskGroup{
		Name: &groupName1,
	}

	groups = append(groups, taskGtoup)

	policy2 := &structs.GroupScalingPolicy{
		GroupName:   "cache2",
		Cooldown:    60,
		Enabled:     true,
		Min:         7500,
		Max:         10000,
		ScaleInMem:  40,
		ScaleInCPU:  40,
		ScaleOutMem: 90,
		ScaleOutCPU: 90,
		UID:         "ELS1",
	}
	scaling.Policies["example"] = append(scaling.Policies["example"], policy2)

	checkOrphanedGroup("example", groups, scaling)
	if !reflect.DeepEqual(scaling, expected) {
		t.Fatalf("expected \n%#v\n\n, got \n\n%#v\n\n", expected, scaling)
	}
}

func exampleJobScalingPolicies() *structs.JobScalingPolicies {
	scaling := &structs.JobScalingPolicies{
		Policies: make(map[string][]*structs.GroupScalingPolicy),
	}

	policy := &structs.GroupScalingPolicy{
		GroupName:   "cache",
		Cooldown:    60,
		Enabled:     true,
		Min:         750,
		Max:         1000,
		ScaleInMem:  30,
		ScaleInCPU:  30,
		ScaleOutMem: 80,
		ScaleOutCPU: 80,
		UID:         "ELS1",
	}
	scaling.Policies["example"] = append(scaling.Policies["example"], policy)
	return scaling
}
