package testutil

import (
	"testing"

	"github.com/d3sw/replicator/replicator/structs"
	"github.com/hashicorp/consul/testutil"
)

// MakeClientWithConfig will be written by Eric.
func MakeClientWithConfig(t *testing.T) (*structs.Config, *testutil.TestServer) {

	srv1, err := testutil.NewTestServerConfigT(t, nil)
	if err != nil {
		t.Fatalf("Failed creating consul test server: %v", err)
		return nil, nil
	}

	config := &structs.Config{
		ConsulKeyRoot: "replicator/config",
		Notification:  &structs.Notification{},
	}

	return config, srv1
}
