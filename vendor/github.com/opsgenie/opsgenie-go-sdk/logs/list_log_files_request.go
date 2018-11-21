package logs

import (
	"errors"
	"fmt"
	"net/url"
)

type ListLogFilesRequest struct {
	ApiKey string
	Marker string
	Limit  string
}

func (r *ListLogFilesRequest) GetApiKey() string {
	return r.ApiKey
}

func (r *ListLogFilesRequest) GetMarker() string {
	return r.Marker
}

func (r *ListLogFilesRequest) GenerateUrl() (string, url.Values, error) {
	if r.Marker == "" {
		return "", nil, errors.New("Marker should be provided")
	}

	params := url.Values{}
	if r.Limit != "" {
		params.Add("limit", r.Limit)
	}
	return fmt.Sprintf("/v2/logs/list/%s", r.Marker), params, nil
}
