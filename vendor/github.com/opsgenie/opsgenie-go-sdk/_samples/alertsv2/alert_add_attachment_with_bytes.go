package main

import (
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	"github.com/opsgenie/opsgenie-go-sdk/alertsv2"
	"fmt"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"io/ioutil"
)

func main() {

	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	alertCli, _ := cli.AlertV2()

	// This is for test purpose only. You can provide your file as byte array however you want.
	fileInBytes := getBytesFromFile("filepath")

	request := alertsv2.AddAlertAttachmentRequest{
		AttachmentAlertIdentifier: &alertsv2.AttachmentAlertIdentifier{
			TinyID: "5",
		},
		AttachmentFileContent: fileInBytes,
		AttachmentFileName: "test.jpg",
		User:   "test",
	}
	response, err := alertCli.AttachFile(request)

	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Result: " + response.Result)
	}
}

func getBytesFromFile(path string) []byte {
	fileBytes, err := ioutil.ReadFile(path)

	if err != nil {
		fmt.Println(err.Error())
	}

	return fileBytes
}
