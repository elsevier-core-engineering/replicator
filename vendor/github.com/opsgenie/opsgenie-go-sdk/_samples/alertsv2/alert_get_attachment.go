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

	request := alertsv2.GetAlertAttachmentRequest{
		AttachmentAlertIdentifier: &alertsv2.AttachmentAlertIdentifier{
			TinyID: "4746",
		},
		AttachmentId:"1500294613021000158",
	}
	response, err := alertCli.GetAttachmentFile(request)

	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Attachment Name : " + response.Attachment.Name)
		fmt.Println("Link : " + response.Attachment.DownloadLink)
	}
}
