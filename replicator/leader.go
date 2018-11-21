package replicator

import (
	"github.com/d3sw/replicator/logging"
	"github.com/d3sw/replicator/replicator/structs"
)

const (
	leaderElectionInterval = 10
	leaderLockTimeout      = 12
)

// LeaderCandidate runs the leader election.
type LeaderCandidate struct {
	consulClient structs.ConsulClient

	leader  bool
	key     string
	session string
	ttl     int

	renewChan chan struct{}
}

// newLeaderCandidate creates a new LeaderCandidate.
func newLeaderCandidate(consulClient structs.ConsulClient, key string, ttl int) *LeaderCandidate {
	return &LeaderCandidate{
		consulClient: consulClient,
		key:          key,
		leader:       false,
		ttl:          ttl,
	}
}

// isLeader returns true if the candidate is currently a leader.
func (l *LeaderCandidate) isLeader() bool {
	return l.leader
}

// leaderElection is the main entry in to the Replicator leadership locking
// process and will create a Consul session for use in obtaining the leader
// lock.
func (l *LeaderCandidate) leaderElection() (isLeader bool) {

	// Create our session and start the renew process if the candidate does not
	// currently have one.
	if l.session == "" {

		l.renewChan = make(chan struct{})

		id, err := l.consulClient.CreateSession(l.ttl, l.renewChan)
		if err != nil {
			logging.Error("core/leader: unable to obtain Consul session: %v", err)
			return
		}

		// Store our sessionID.
		l.session = id
	}

	// Attempt to acquire the leadership lock.
	if isLeader = l.consulClient.AcquireLeadership(l.key, &l.session); isLeader {
		logging.Debug("core/leader: currently running as Replicator leader")
		l.leader = true
		return true
	}

	logging.Debug("core/leader: failed to acquire leadership lock")
	return
}

// endCampaign attempts to gracefully remove sessions and locks associated with
// the replicator leadership locking, allowing other daemons to pick up the lock
// without having to wait for the TTL to expire.
func (l *LeaderCandidate) endCampaign() {
	logging.Info("core/leader: gracefully cleaning up Consul sessions and locks")

	l.releaseSession()

	if l.leader {
		l.consulClient.ResignLeadership(l.key, l.session)
	}
}

// releaseSession closes the renewChan therefore telling the renewSession
// process to try and destroy the session.
func (l *LeaderCandidate) releaseSession() {
	if l.renewChan != nil {
		close(l.renewChan)
	}
}
