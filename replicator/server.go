package replicator

import (
	"errors"
	"fmt"
	"net"
	"net/rpc"
	"reflect"
	"time"

	"github.com/d3sw/replicator/logging"
	"github.com/d3sw/replicator/replicator/structs"
)

// Server is the Replicator server that is responsible for running the API and
// all scaling tasks.
type Server struct {
	// candaidate is our LeaderCandidate for the runner instance.
	candidate *LeaderCandidate

	// config is the Config that created this Runner. It is used internally to
	// construct other objects and pass data.
	config *structs.Config

	// endpoints represents the Replicator API endpoints.
	endpoints endpoints

	rpcAdvertise net.Addr
	rpcListener  net.Listener
	rpcServer    *rpc.Server

	shutdown     bool
	shutdownChan chan struct{}
}

// endpoints represents the Replicator API endpoints.
type endpoints struct {
	Status *Status
}

type inmemCodec struct {
	method string
	args   interface{}
	reply  interface{}
	err    error
}

// NewServer is the main entry point into Replicator and launches processes based
// on the configuration.
func NewServer(config *structs.Config) (*Server, error) {

	s := &Server{
		config:       config,
		rpcServer:    rpc.NewServer(),
		shutdownChan: make(chan struct{}),
	}

	// Setup our LeaderCandidate object for leader elections and session renewal.
	leaderKey := s.config.ConsulKeyRoot + "/" + "leader"
	s.candidate = newLeaderCandidate(s.config.ConsulClient, leaderKey,
		leaderLockTimeout)
	go s.leaderTicker()

	jobScalingPolicy := newJobScalingPolicy()

	if !s.config.ClusterScalingDisable || !s.config.JobScalingDisable {
		// Setup our JobScalingPolicy Watcher and start running this.
		go s.config.NomadClient.JobWatcher(jobScalingPolicy)
	}

	if !s.config.ClusterScalingDisable {
		// Setup the node registry and initiate worker pool and node discovery.
		nodeRegistry := structs.NewNodeRegistry()
		go s.config.NomadClient.NodeWatcher(nodeRegistry, s.config)

		// Launch our cluster scaling main ticker function
		go s.clusterScalingTicker(nodeRegistry, jobScalingPolicy)
	}

	// Launch our job scaling main ticker function
	if !s.config.JobScalingDisable {
		go s.jobScalingTicker(jobScalingPolicy)
	}

	if err := s.setupRPC(); err != nil {
		s.Shutdown()
		return nil, fmt.Errorf("failed to start RPC layer: %v", err)
	}

	// Start the RPC listeners
	go s.listen()
	logging.Info("core/server: the RPC server has started and is listening at %v", s.config.RPCAddr)

	return s, nil
}

// Shutdown halts the execution of the server.
func (s *Server) Shutdown() {
	s.candidate.endCampaign()

	// Shutdown the RPC listener.
	if s.rpcListener != nil {
		logging.Info("core/server: shutting down RPC server at %v", s.rpcListener.Addr())
		s.rpcListener.Close()
	}

	close(s.shutdownChan)
}

func (s *Server) leaderTicker() {
	ticker := time.NewTicker(
		time.Second * time.Duration(leaderElectionInterval),
	)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Perform the leadership locking and continue if we have confirmed that
			// we are running as the replicator leader.
			s.candidate.leaderElection()
		case <-s.shutdownChan:
			return
		}
	}
}

func (s *Server) jobScalingTicker(jobPol *structs.JobScalingPolicies) {
	ticker := time.NewTicker(
		time.Second * time.Duration(s.config.JobScalingInterval),
	)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if s.candidate.isLeader() && len(jobPol.Policies) > 0 {
				s.asyncJobScaling(jobPol)
			}
		case <-s.shutdownChan:
			return
		}
	}
}

func (s *Server) clusterScalingTicker(nodeReg *structs.NodeRegistry, jobPol *structs.JobScalingPolicies) {
	ticker := time.NewTicker(
		time.Second * time.Duration(s.config.ClusterScalingInterval),
	)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if s.candidate.isLeader() && len(nodeReg.WorkerPools) > 0 {
				err := s.nodeProtectionCheck(nodeReg)
				if err != nil {
					logging.Error("core/runner: an error occurred while trying to "+
						"protect the node running the Replicator leader: %v", err)
				}

				s.asyncClusterScaling(nodeReg, jobPol)

			}
		case <-s.shutdownChan:
			return
		}
	}
}

// setupRPC is used to setup our endpoints and register the handlers as well as
// setup the RPC listener.
func (s *Server) setupRPC() error {

	s.endpoints.Status = &Status{s}
	s.rpcServer.Register(s.endpoints.Status)

	list, err := net.ListenTCP("tcp", s.config.RPCAddr)
	if err != nil {
		return err
	}
	s.rpcListener = list

	s.rpcAdvertise = s.rpcListener.Addr()

	// Verify that we have a usable advertise address
	addr, ok := s.rpcAdvertise.(*net.TCPAddr)
	if !ok {
		list.Close()
		return fmt.Errorf("RPC advertise address is not a TCP Address: %v", addr)
	}
	if addr.IP.IsUnspecified() {
		list.Close()
		return fmt.Errorf("RPC advertise address is not advertisable: %v", addr)
	}

	return nil
}

func (i *inmemCodec) ReadRequestHeader(req *rpc.Request) error {
	req.ServiceMethod = i.method
	return nil
}

func (i *inmemCodec) ReadRequestBody(args interface{}) error {
	return nil
}

func (i *inmemCodec) WriteResponse(resp *rpc.Response, reply interface{}) error {
	if resp.Error != "" {
		i.err = errors.New(resp.Error)
		return nil
	}
	sourceValue := reflect.Indirect(reflect.Indirect(reflect.ValueOf(reply)))
	dst := reflect.Indirect(reflect.Indirect(reflect.ValueOf(i.reply)))
	dst.Set(sourceValue)
	return nil
}

func (i *inmemCodec) Close() error {
	return nil
}

// RPC is used to make an RPC call.
func (s *Server) RPC(method string, reply interface{}) error {
	codec := &inmemCodec{
		method: method,
		reply:  reply,
	}
	if err := s.rpcServer.ServeRequest(codec); err != nil {
		return err
	}
	return codec.err
}
