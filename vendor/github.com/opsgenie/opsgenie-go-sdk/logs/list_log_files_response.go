package logs

type ListLogFilesResponse struct {
	RequestID      string
	ResponseTime   float32
	RateLimitState string
	Marker         string `json:"marker"`
	Logs           []Log  `json:"data"`
}

// SetRequestID sets identifier of request.
func (rm *ListLogFilesResponse) SetRequestID(requestID string) {
	rm.RequestID = requestID
}

// SetResponseTime sets request execution time.
func (rm *ListLogFilesResponse) SetResponseTime(responseTime float32) {
	rm.ResponseTime = responseTime
}

// SetRateLimitState sets state of rate limit.
func (rm *ListLogFilesResponse) SetRateLimitState(state string) {
	rm.RateLimitState = state
}
