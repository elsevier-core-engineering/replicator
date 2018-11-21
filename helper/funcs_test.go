package helper

import (
	"testing"

	"github.com/d3sw/replicator/replicator/structs"
)

func TestHelper_FindIpP(t *testing.T) {

	input := "10.0.0.10:4646"
	expected := "10.0.0.10"

	ip := FindIP(input)
	if ip != expected {
		t.Fatalf("expected %s got %s", expected, ip)
	}
}

func TestHelper_Max(t *testing.T) {

	expected := 13.12

	max := Max(13.12, 2.01, 6.4, 13.11, 1.01, 0.11)
	if max != expected {
		t.Fatalf("expected %v got %v", expected, max)
	}
}

func TestHelper_StringInList(t *testing.T) {
	type stringTest struct {
		input    string
		expected bool
	}

	var stringTests = []stringTest{
		{"goo", false}, {"foo", true},
	}

	list := []string{"foo", "bar"}

	for _, test := range stringTests {
		actual := StringInSlice(test.input, list)

		if actual != test.expected {
			t.Fatalf("expected %v got %v", test.expected, actual)
		}
	}
}

func TestHelper_Min(t *testing.T) {

	expected := 1.01

	min := Min(13.12, 2.01, 6.4, 13.11, 1.01, 1.02)
	if min != expected {
		t.Fatalf("expected %v got %v", expected, min)
	}
}

func TestHelper_HasObjectChanged(t *testing.T) {
	policyA := &structs.GroupScalingPolicy{
		GroupName:   "core-engineering",
		Enabled:     true,
		Max:         10,
		Min:         1,
		ScaleInMem:  10,
		ScaleInCPU:  20,
		ScaleOutMem: 90,
		ScaleOutCPU: 90,
	}
	policyB := &structs.GroupScalingPolicy{
		GroupName:   "core-engineering",
		Enabled:     true,
		Max:         10,
		Min:         1,
		ScaleInMem:  10,
		ScaleInCPU:  20,
		ScaleOutMem: 90,
		ScaleOutCPU: 90,
	}

	change, err := HasObjectChanged(policyA, policyB)
	if err != nil {
		t.Fatal(err)
	}

	if change {
		t.Fatalf("expected false but got %v", change)
	}

	policyB.ScaleDirection = "Out"
	change, err = HasObjectChanged(policyA, policyB)
	if err != nil {
		t.Fatal(err)
	}

	if change {
		t.Fatalf("expected false but got %v", change)
	}

	policyB.ScaleInMem = 20
	change, err = HasObjectChanged(policyA, policyB)
	if err != nil {
		t.Fatal(err)
	}

	if !change {
		t.Fatalf("expected true but got %v", change)
	}

}

func TestHelper_ParseMetaConfig(t *testing.T) {
	metaKeys := make(map[string]string)

	// These are our required key for Replicator
	requiredKeys := []string{
		"replicator_cooldown",
		"replicator_enabled",
		"replicator_min",
		"replicator_max",
		"replicator_scalein_mem",
		"replicator_scalein_cpu",
		"replicator_scaleout_mem",
		"replicator_scaleout_cpu",
	}

	zeroKeyReturn := ParseMetaConfig(metaKeys, requiredKeys)
	if len(zeroKeyReturn) != 8 {
		t.Fatalf("expected 8 required keys to be returned, got %v", len(zeroKeyReturn))
	}

	metaKeys["replicator_cooldown"] = "60"
	metaKeys["replicator_enabled"] = "true"
	metaKeys["replicator_max"] = "1000"
	metaKeys["replicator_min"] = "750"
	metaKeys["replicator_scalein_mem"] = "40"
	metaKeys["replicator_scalein_cpu"] = "40"
	metaKeys["replicator_scaleout_mem"] = "90"

	partialKeyReturn := ParseMetaConfig(metaKeys, requiredKeys)
	if len(partialKeyReturn) != 1 {
		t.Fatalf("expected 1 required keys to be returned, got %v", len(partialKeyReturn))
	}

	metaKeys["replicator_scaleout_cpu"] = "90"

	allKeysReturn := ParseMetaConfig(metaKeys, requiredKeys)
	if len(allKeysReturn) != 0 {
		t.Fatalf("expected 0 required keys to be returned, got %v", len(allKeysReturn))
	}
}
