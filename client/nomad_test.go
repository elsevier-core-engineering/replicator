package client

import (
	"testing"

	"github.com/d3sw/replicator/replicator/structs"
)

func TestNomad_NewNomadClient(t *testing.T) {
	addr := "http://nomad.ce.systems:4646"
	config := "nomad.ce.systems"
	token := "change_me"

	_, err := NewNomadClient(addr, token, config)
	if err != nil {
		t.Fatalf("error creating Nomad client %s", err)
	}
}

func TestNomad_CalculateUsage(t *testing.T) {
	node1 := &structs.NodeAllocation{
		UsedCapacity: structs.AllocationResources{
			CPUMHz:   600,
			DiskMB:   1280,
			MemoryMB: 2048,
		},
	}
	node2 := &structs.NodeAllocation{
		UsedCapacity: structs.AllocationResources{
			CPUMHz:   600,
			DiskMB:   1280,
			MemoryMB: 2048,
		},
	}

	nodes := make([]*structs.NodeAllocation, 2)
	nodes[0] = node1
	nodes[1] = node2

	cap := &structs.ClusterCapacity{
		UsedCapacity: structs.AllocationResources{
			CPUMHz:   1200,
			DiskMB:   2560,
			MemoryMB: 4096,
		},
		TotalCapacity: structs.AllocationResources{
			CPUMHz:   2400,
			DiskMB:   5120,
			MemoryMB: 8192,
		},
		NodeAllocations: nodes,
	}

	var percentage = 50.00
	CalculateUsage(cap)

	if cap.UsedCapacity.CPUPercent != percentage {
		t.Fatalf("expected cluster CPU percentage %v but got %v", percentage, cap.UsedCapacity.CPUPercent)
	}
	if cap.UsedCapacity.DiskPercent != percentage {
		t.Fatalf("expected cluster Disk percentage %v but got %v", percentage, cap.UsedCapacity.DiskPercent)
	}
	if cap.UsedCapacity.MemoryPercent != percentage {
		t.Fatalf("expected cluster Memory percentage %v but got %v", percentage, cap.UsedCapacity.MemoryPercent)
	}
	for _, node := range cap.NodeAllocations {
		if node.UsedCapacity.CPUPercent != percentage {
			t.Fatalf("expected node CPU percentage %v but got %v", percentage, node.UsedCapacity.CPUPercent)
		}
		if node.UsedCapacity.DiskPercent != percentage {
			t.Fatalf("expected node CPU percentage %v but got %v", percentage, node.UsedCapacity.DiskPercent)
		}
		if node.UsedCapacity.MemoryPercent != percentage {
			t.Fatalf("expected node CPU percentage %v but got %v", percentage, node.UsedCapacity.MemoryPercent)
		}
	}
}

func TestNomad_MostUtilizedResource(t *testing.T) {
	client := &nomadClient{}

	cpuCap := &structs.ClusterCapacity{
		UsedCapacity: structs.AllocationResources{
			CPUPercent:    11,
			MemoryPercent: 10,
			DiskPercent:   9,
		},
	}
	memCap := &structs.ClusterCapacity{
		UsedCapacity: structs.AllocationResources{
			CPUPercent:    90,
			MemoryPercent: 91,
			DiskPercent:   89,
		},
	}
	diskCap := &structs.ClusterCapacity{
		UsedCapacity: structs.AllocationResources{
			CPUPercent:    31,
			MemoryPercent: 32,
			DiskPercent:   33,
		},
	}
	zeroCap := &structs.ClusterCapacity{
		UsedCapacity: structs.AllocationResources{
			CPUPercent:    0,
			MemoryPercent: 0,
			DiskPercent:   0,
		},
	}

	client.MostUtilizedResource(cpuCap)
	client.MostUtilizedResource(memCap)
	client.MostUtilizedResource(diskCap)
	client.MostUtilizedResource(zeroCap)

	if cpuCap.ScalingMetric.Type != ScalingMetricProcessor {
		t.Fatalf("expected scaling metric %v but got %v", ScalingMetricProcessor, cpuCap.ScalingMetric)
	}
	if memCap.ScalingMetric.Type != ScalingMetricMemory {
		t.Fatalf("expected scaling metric %v but got %v", ScalingMetricMemory, memCap.ScalingMetric)
	}
	if diskCap.ScalingMetric.Type != ScalingMetricDisk {
		t.Fatalf("expected scaling metric %v but got %v", ScalingMetricDisk, diskCap.ScalingMetric)
	}
	if zeroCap.ScalingMetric.Type != ScalingMetricNone {
		t.Fatalf("expected scaling metric %v but got %v", ScalingMetricNone, zeroCap.ScalingMetric)
	}
}

func TestNomad_MostUtilizedGroupResource(t *testing.T) {
	client := &nomadClient{}

	memGSP := &structs.GroupScalingPolicy{
		Tasks: structs.TaskAllocation{
			Resources: structs.AllocationResources{
				CPUPercent:    9.98,
				MemoryPercent: 9.99,
			},
		},
	}
	cpuGSP := &structs.GroupScalingPolicy{
		Tasks: structs.TaskAllocation{
			Resources: structs.AllocationResources{
				CPUPercent:    11,
				MemoryPercent: 10,
			},
		},
	}

	client.MostUtilizedGroupResource(memGSP)
	client.MostUtilizedGroupResource(cpuGSP)

	if memGSP.ScalingMetric != ScalingMetricMemory {
		t.Fatalf("expected scaling metric %v but got %v", ScalingMetricMemory, memGSP.ScalingMetric)
	}
	if cpuGSP.ScalingMetric != ScalingMetricProcessor {
		t.Fatalf("expected scaling metric %v but got %v", ScalingMetricProcessor, cpuGSP.ScalingMetric)
	}
}

func TestNomad_MaxAllowedClusterUtilization(t *testing.T) {
	nodeFaultTolerance := 1
	scaleIn := true

	cap := &structs.ClusterCapacity{
		ScalingMetric: structs.ScalingMetric{
			Type: ScalingMetricMemory,
		},
		NodeCount: 4,
		TaskAllocation: structs.AllocationResources{
			MemoryMB: 2048,
			CPUMHz:   2400,
		},
		UsedCapacity: structs.AllocationResources{
			MemoryMB: 2048,
			CPUMHz:   2400,
		},
		TotalCapacity: structs.AllocationResources{
			MemoryMB: 8192,
			CPUMHz:   9600,
		},
	}
	memCap := MaxAllowedClusterUtilization(cap, nodeFaultTolerance, scaleIn)

	cap.ScalingMetric.Type = ScalingMetricProcessor
	cpuCap := MaxAllowedClusterUtilization(cap, nodeFaultTolerance, scaleIn)

	if memCap != 2048 {
		t.Fatalf("expected max memory utilization of 2048 but got %v", memCap)
	}
	if cpuCap != 2400 {
		t.Fatalf("expected max memory utilization of 2048 but got %v", cpuCap)
	}
}
