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

	request := alertsv2.EscalateToNextRequest{
		Identifier: &alertsv2.Identifier{
			TinyID: "2",
		},
		User:   "test",
		Source: "Source",
		Note:   "Note",
		Escalation: alertsv2.Escalation{
			Name: "escalationName",
		},
	}
	response, err := alertCli.EscalateToNext(request)

	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("RequestId: " + response.RequestID)
	}
}
