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

	request := alertsv2.ListAlertAttachmentRequest{
		AttachmentAlertIdentifier: &alertsv2.AttachmentAlertIdentifier{
			TinyID: "4746",
		},
	}
	response, err := alertCli.ListAlertAttachments(request)

	if err != nil {
		fmt.Println(err.Error())
	} else {
		for _, attachment := range response.AlertAttachments {
			fmt.Printf("AttachmentName : %s\n" , attachment.Name)
			fmt.Printf("AttachmentID : %d\n" , attachment.Id)
		}
	}
}
