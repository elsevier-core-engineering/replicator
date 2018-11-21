package logs

type GenerateLogFileDownloadResponse struct {
	RequestID      string
	ResponseTime   float32
	RateLimitState string
	DownloadLink   string
}

func (rm *GenerateLogFileDownloadResponse) SetRequestID(requestID string) {
	rm.RequestID = requestID
}

func (rm *GenerateLogFileDownloadResponse) SetResponseTime(responseTime float32) {
	rm.ResponseTime = responseTime
}

func (rm *GenerateLogFileDownloadResponse) SetRateLimitState(state string) {
	rm.RateLimitState = state
}