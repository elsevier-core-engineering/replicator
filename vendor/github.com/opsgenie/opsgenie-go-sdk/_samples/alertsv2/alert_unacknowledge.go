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

	identifier := alertsv2.Identifier{
		TinyID: "2",
	};

	unackRequest := alertsv2.UnacknowledgeRequest{
		Identifier: &identifier,
		User:       "test",
		Source:     "Source",
		Note:       "Note",
	}

	response, _ := alertCli.Unacknowledge(unackRequest)
	fmt.Println("RequestID: " + response.RequestID)
}