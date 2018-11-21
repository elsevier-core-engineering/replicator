package main

import (
	"fmt"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	"github.com/opsgenie/opsgenie-go-sdk/logs"

	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	logCli, _ := cli.Log()

	response, err := logCli.LogFileDownloadLink(logs.GenerateLogFileDownloadRequest{
		Filename: "2018-10-31-11-53-51.json",
	})

	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Printf("DownloadLink: %s", response)
	}
}
