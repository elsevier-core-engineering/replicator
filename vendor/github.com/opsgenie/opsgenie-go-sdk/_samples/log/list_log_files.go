package main

import (
	"fmt"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/opsgenie/opsgenie-go-sdk/logs"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	logCli, _ := cli.Log()

	response, err := logCli.ListLogFiles(logs.ListLogFilesRequest{
		Marker: "2018-10-24-07-30-00",
		Limit:  "100",
	})

	if err != nil {
		fmt.Println(err.Error())
	} else {
		for _, logg := range response.Logs {
			fmt.Printf(
				"Filename: %s, Date: %d, Size: %d\n", logg.Filename, logg.Date, logg.Size,
			)
		}
	}
}
