package replicator

import (
	"reflect"
	"testing"

	"github.com/d3sw/replicator/client"
)

func TestLeader_newLeaderCandidate(t *testing.T) {
	consul, _ := client.NewConsulClient("172.0.0.1", "")
	key := "replicator/config/leader"
	ttl := 60

	expected := &LeaderCandidate{
		consulClient: consul,
		key:          key,
		leader:       false,
		ttl:          ttl,
	}

	candidate := newLeaderCandidate(consul, key, ttl)

	if !reflect.DeepEqual(candidate, expected) {
		t.Fatalf("expected \n%#v\n\n, got \n\n%#v\n\n", expected, candidate)
	}
}

func TestLeader_isLeader(t *testing.T) {
	consul, _ := client.NewConsulClient("172.0.0.1", "")
	key := "replicator/config/leader"
	ttl := 60

	l := &LeaderCandidate{
		consulClient: consul,
		key:          key,
		leader:       false,
		ttl:          ttl,
	}

	if l.isLeader() {
		t.Fatal("expected isLeadrer to answer false but got true")
	}
	l.leader = true
	if !l.isLeader() {
		t.Fatal("expected isLeadrer to answer true but got false")
	}
}
