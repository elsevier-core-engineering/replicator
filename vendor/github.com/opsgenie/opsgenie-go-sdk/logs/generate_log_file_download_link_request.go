package logs

import (
	"errors"
	"fmt"
	"net/url"
)

type GenerateLogFileDownloadRequest struct {
	ApiKey   string
	Filename string
}

func (r *GenerateLogFileDownloadRequest) GetApiKey() string {
	return r.ApiKey
}

func (r *GenerateLogFileDownloadRequest) GetFilename() string {
	return r.Filename
}

func (r *GenerateLogFileDownloadRequest) GenerateUrl() (string, url.Values, error) {
	if r.Filename == "" {
		return "", nil, errors.New("Filename should be provided")
	}
	return fmt.Sprintf("/v2/logs/download/%s", r.Filename), nil, nil
}
