package client

import (
	"reflect"
	"testing"

	"github.com/d3sw/replicator/replicator/structs"
	"github.com/d3sw/replicator/testutil"
)

func TestClient_StateTracking(t *testing.T) {
	t.Parallel()

	c, s := testutil.MakeClientWithConfig(t)
	defer s.Stop()

	consul, _ := NewConsulClient(s.HTTPAddr, "")
	c.ConsulClient = consul

	returnState := &structs.ScalingState{}
	returnState.StatePath = c.ConsulKeyRoot + "/state/nodes/example-pool"

	expected := &structs.ScalingState{
		FailsafeMode: false,
		FailureCount: 0,
		StatePath:    c.ConsulKeyRoot + "/state/nodes/example-pool",
	}

	err := c.ConsulClient.PersistState(expected)
	if err != nil {
		t.Fatalf("error writing state to Consul: %v", err)
	}

	c.ConsulClient.ReadState(returnState, true)
	expected.LastUpdated = returnState.LastUpdated
	if !reflect.DeepEqual(returnState, expected) {
		t.Fatalf("expected \n%#v\n\n, got \n\n%#v\n\n", expected, returnState)
	}

}
