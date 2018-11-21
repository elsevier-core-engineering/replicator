package client

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"

	metrics "github.com/armon/go-metrics"
	consul "github.com/hashicorp/consul/api"

	"github.com/d3sw/replicator/logging"
	"github.com/d3sw/replicator/replicator/structs"
)

// The client object is a wrapper to the Consul client provided by the Consul
// API library.
type consulClient struct {
	consul *consul.Client
}

// NewConsulClient is used to construct a new Consul client using the default
// configuration and supporting the ability to specify a Consul API address
// endpoint in the form of address:port.
func NewConsulClient(addr, token string) (structs.ConsulClient, error) {
	// TODO (e.westfall): Add a quick health check call to an API endpoint to
	// validate connectivity or return an error back to the caller.
	config := consul.DefaultConfig()
	addrArr := strings.Split(addr, "://")

	// Checking scheme of consul endpoint based on address
	if len(addrArr) > 1 {
		config.Scheme = addrArr[0]
		config.Address = addrArr[1]
	} else {
		config.Address = addrArr[0]
	}

	if token != "" {
		config.Token = token
	}

	c, err := consul.NewClient(config)
	if err != nil {
		// TODO (e.westfall): Raise error here.
		return nil, err
	}

	return &consulClient{consul: c}, nil
}

// GetLeaderInfo is used to inspect the leadership KV and session to provide
// details of the Replicator agent currently holding leadership.
func (c *consulClient) GetLeaderInfo(info *structs.LeaderResponse, key *string, session string) error {

	if session == "" {
		k, _, err := c.consul.KV().Get(*key, nil)
		if err != nil {
			return err
		}
		session = k.Session
	}

	s, _, err := c.consul.Session().Info(session, nil)
	if err != nil {
		return err
	}

	info.NodeID = s.Node
	info.SessionID = s.ID

	return nil
}

// CreateSession creates a Consul session for use in the Leadership locking
// process and will spawn off the renewing of the session in order to ensure
// leadership can be maintained.
func (c *consulClient) CreateSession(ttl int, stopCh chan struct{}) (id string, err error) {

	entry := &consul.SessionEntry{
		TTL:  fmt.Sprintf("%vs", ttl),
		Name: "replicator_leader_lock",
	}

	// Create the key session.
	logging.Debug("client/consul: obtaining Consul session with %vs TTL", ttl)
	resp, _, err := c.consul.Session().Create(entry, nil)
	if err != nil {
		return "", err
	}

	// Spawn off to continue renewing our session.
	c.renewSession(entry.TTL, resp, stopCh)

	return resp, nil
}

// LoadPoolConfig retrieves fallback worker pool configuration from Consul.
func (c *consulClient) LoadPoolConfig(key string) (map[string]string, error) {
	// Attempt to read the pool configuration key, if present.
	config, _, err := c.consul.KV().Get(key, nil)
	if err != nil || config == nil {
		return nil, fmt.Errorf("unable to find or read pool configuration data "+
			"at path %v: %v", key, err)
	}

	// Marshal result into map object.
	result := make(map[string]string)
	err = json.Unmarshal(config.Value, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to process pool configuration data read "+
			"from path %v: %v", key, err)
	}

	return result, nil
}

// AcquireLeadership attempts to acquire a Consul leadersip lock using the
// provided session. If the lock is already taken this will return false in
// a show that there is already a leader.
func (c *consulClient) AcquireLeadership(key string, session *string) (acquired bool) {

	// Attempt to inspect the leadership key if it is available and present.
	k, _, err := c.consul.KV().Get(key, nil)

	if err != nil {
		logging.Error("client/consul: unable to read the leader key at %s", key)
		return false
	}

	// Check we have a valid session.
	s, _, err := c.consul.Session().Info(*session, nil)
	if err != nil {
		logging.Error("client/consul: unable to read the leader key at %s", key)
		return false
	}

	// If the session is not valid, set our state to default
	if s == nil {
		logging.Error("client/consul: the Consul session %s has expired, revoking from Replicator", *session)
		*session = ""
		return false
	}

	// On a fresh cluster the KV might not exist yet, so we need to check for nil
	// return. If the leadership lock is tied to our session then we can exit and
	// confirm we are running as the replicator leader without having to make on
	// further calls.
	if k != nil && k.Session == *session {
		return true
	}

	kp := &consul.KVPair{
		Key:     key,
		Session: *session,
	}

	logging.Debug("client/consul: attempting to acquire leadership lock at %s", key)
	resp, _, err := c.consul.KV().Acquire(kp, nil)

	if err != nil {
		logging.Error("client/consul: issue requesting leadership lock: %v", err)
		return false
	}

	// We have successfully obtained the leadership and can now be considered as
	// the replicator leader.
	if resp {
		logging.Info("client/consul: leadership lock successfully obtained at %s", key)
		metrics.IncrCounter([]string{"cluster", "leadership", "election"}, 1)
		return true
	}

	return false
}

// ResignLeadership attempts to remove the leadership lock upon shutdown of the
// replicator daemon. If this is unsuccessful there is not too much we can do
// therefore there is no return.
func (c *consulClient) ResignLeadership(key, session string) {

	kp := &consul.KVPair{
		Key:     key,
		Session: session,
	}

	resp, _, err := c.consul.KV().Release(kp, nil)
	if err != nil {
		logging.Error("client/consul: unable to successfully release leadership lock: %v", err)
		return
	}

	// If we get a successful response we should log it.
	if resp {
		logging.Info("client/consul: the leadership lock has now been released")
	}

	return
}

// renewSession is used for renewing a Consul session and accepts a channel
// within which a signal can be sent which will stop the renawl process and
// attempt to clean up the session.
func (c *consulClient) renewSession(ttl string, session string, renewChan chan struct{}) {

	sessionDestroyAttempts := 0
	maxSessionDestroyAttempts := 5

	parsedTTL, err := time.ParseDuration(ttl)
	if err != nil {
		return
	}

	go func() {
		for {
			select {
			case <-time.After(parsedTTL / 2):
				entry, _, err := c.consul.Session().Renew(session, nil)
				if err != nil {
					logging.Error("client/consul: unable to renew the Consul session %s: %v", session, err)
					runtime.Goexit()
				}
				if entry == nil {
					return
				}

				// Consul may return a TTL value higher than the one specified during
				// session creation. This indicates the server is under high load and
				// is requesting clients renew less often. If this happens we need to
				// ensure we track the new TTL.
				parsedTTL, _ = time.ParseDuration(entry.TTL)
				logging.Debug("client/consul: the Consul session %s has been renewed", session)

			case <-renewChan:
				_, err := c.consul.Session().Destroy(session, nil)
				if err == nil {
					logging.Info("client/consul: the Consul session %s has been released", session)
					return
				}

				if sessionDestroyAttempts >= maxSessionDestroyAttempts {
					logging.Error("client/consul: unable to successfully destroy the Consul session %s", session)
					return
				}

				// We can't destroy the session so we will wait and attempt again until
				// we hit the threshold.
				sessionDestroyAttempts++
				time.Sleep(parsedTTL)
			}
		}
	}()
}
