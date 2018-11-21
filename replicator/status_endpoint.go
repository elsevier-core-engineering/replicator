package replicator

import (
	"github.com/d3sw/replicator/replicator/structs"
)

// Status endpoint is used to get information on the server status.
type Status struct {
	srv *Server
}

// Leader gets information regarding the Replicator instance which is holding
// leadership.
func (s *Status) Leader(args interface{}, reply *structs.LeaderResponse) error {

	var session string

	if s.srv.candidate.leader {
		session = s.srv.candidate.session
	} else {
		session = ""
	}

	err := s.srv.config.ConsulClient.GetLeaderInfo(reply, &s.srv.candidate.key, session)
	if err != nil {
		return err
	}

	return nil
}
