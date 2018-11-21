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

	request := alertsv2.ListAlertNotesRequest{
		Identifier: &alertsv2.Identifier{
			TinyID: "2",
		},
		Order:     "asc",
		Direction: alertsv2.Next,
		Offset:    "1",
		Limit:     2,
	}
	response, err := alertCli.ListAlertNotes(request)

	if err != nil {
		fmt.Println(err.Error())
	} else {
		for _, note := range response.AlertNotes {
			fmt.Println(note.Note)
		}
	}
}