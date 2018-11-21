package api

import "github.com/d3sw/replicator/replicator/structs"

// Status is used to query all status related endpoints.
type Status struct {
	client *Client
}

// Status returns a handle on the status related endpoints.
func (c *Client) Status() *Status {
	return &Status{client: c}
}

// Leader is used to query information regarding the current Replicator leader.
func (s *Status) Leader() (structs.LeaderResponse, error) {
	var resp structs.LeaderResponse

	err := s.client.query("/v1/status/leader", &resp)
	if err != nil {
		return resp, err
	}

	return resp, nil
}
