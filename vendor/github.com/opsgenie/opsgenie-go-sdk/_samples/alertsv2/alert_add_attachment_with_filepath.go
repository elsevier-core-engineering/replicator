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

	request := alertsv2.AddAlertAttachmentRequest{
		AttachmentAlertIdentifier: &alertsv2.AttachmentAlertIdentifier{
			TinyID: "5",
		},
		AttachmentFilePath: constants.PathToFile,
		User:   "test",
	}
	response, err := alertCli.AttachFile(request)

	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Result: " + response.Result)
	}
}
