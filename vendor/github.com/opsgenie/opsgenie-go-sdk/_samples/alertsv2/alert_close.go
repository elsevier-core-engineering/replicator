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

	closeRequest := alertsv2.CloseRequest{
		Identifier: &identifier,
		User:       "test",
		Source:     "Source",
		Note:       "Note",
	}

	response, err := alertCli.Close(closeRequest)

	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("RequestID: " + response.RequestID)
	}
}