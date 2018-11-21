package main

import (
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	"github.com/opsgenie/opsgenie-go-sdk/alertsv2"
	"fmt"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	alertCli, _ := cli.AlertV2()

	request := alertsv2.GetAsyncRequestStatusRequest{
		RequestID: "request_id",
	}

	response, _ := alertCli.GetAsyncRequestStatus(request)

	status := response.Status

	fmt.Println("AlertId:" + status.AlertID)
	fmt.Println("Status: "  + status.Status)
	fmt.Println("Alias: " + status.Alias)
}
