package client

import (
	"github.com/opsgenie/opsgenie-go-sdk/logs"
)

// OpsGenieLogClient is the data type to make Customer Log rule API requests.
type OpsGenieLogClient struct {
	RestClient
}

// SetOpsGenieClient sets the embedded OpsGenieClient type of the OpsGenieScheduleRotationV2Client.
func (cli *OpsGenieLogClient) SetOpsGenieClient(ogCli OpsGenieClient) {
	cli.OpsGenieClient = ogCli
}

func (cli *OpsGenieLogClient) ListLogFiles(req logs.ListLogFilesRequest) (*logs.ListLogFilesResponse, error) {
	var response logs.ListLogFilesResponse
	err := cli.sendGetRequest(&req, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (cli *OpsGenieLogClient) LogFileDownloadLink(req logs.GenerateLogFileDownloadRequest) (string, error) {
	path, params, err := req.GenerateUrl()
	if err != nil {
		return "", err
	}

	request := cli.buildGetRequest(cli.generateFullPathWithParams(path, params), nil)
	cli.setApiKey(&request, req.GetApiKey())
	httpResponse, err := cli.sendRequest(request)
	if err != nil {
		return "", err
	}
	defer httpResponse.Body.Close()
	return httpResponse.Body.ToString()
}
