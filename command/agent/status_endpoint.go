package agent

import (
	"net/http"

	"github.com/d3sw/replicator/replicator/structs"
)

// StatusLeaderRequest is used to perform the Status.Leader API request.
func (s *HTTPServer) StatusLeaderRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	if req.Method != "GET" {
		return nil, CodedError(405, ErrInvalidMethod)
	}

	var leader structs.LeaderResponse
	if err := s.agent.RPC("Status.Leader", &leader); err != nil {
		return nil, err
	}
	return leader, nil
}
