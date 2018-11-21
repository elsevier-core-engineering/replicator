package main

import (
	"fmt"

	alerts "github.com/opsgenie/opsgenie-go-sdk/alerts"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	samples "github.com/opsgenie/opsgenie-go-sdk/_samples"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	alertCli, cliErr := cli.Alert()

	if cliErr != nil {
		panic(cliErr)
	}

	// create the alert
	req := alerts.CreateAlertRequest{Message: samples.RandStringWithPrefix("Test", 8)}
	response, alertErr := alertCli.Create(req)

	if alertErr != nil {
		panic(alertErr)
	}

	fmt.Printf("message: %s\n", response.Message)
	fmt.Printf("alert id: %s\n", response.AlertID)
	fmt.Printf("status: %s\n", response.Status)
	fmt.Printf("code: %d\n", response.Code)

	addnotereq := alerts.AddNoteAlertRequest{}
	// add alert ten notes
	for i := 0; i < 10; i++ {
		addnotereq.ID = response.AlertID
		addnotereq.Note = samples.RandString(45)
		addnoteresp, alertErr := alertCli.AddNote(addnotereq)
		if alertErr != nil {
			panic(alertErr)
		}
		fmt.Printf("[Add note] %s %d\n", addnoteresp.Status, addnoteresp.Code)
	}
	listNotesReq := alerts.ListAlertNotesRequest{ID: response.AlertID}
	listNotesResponse, alertErr := alertCli.ListNotes(listNotesReq)
	if alertErr != nil {
		panic(alertErr)
	}

	alertNotes := listNotesResponse.Notes

	fmt.Printf("Last key: %s\n", listNotesResponse.LastKey)
	fmt.Printf("Notes:\n")
	fmt.Printf("------\n")

	for _, note := range alertNotes {
		fmt.Printf("Note: %s\n", note.Note)
		fmt.Printf("Owner: %s\n", note.Owner)
		fmt.Printf("Created at: %d\n", note.CreatedAt)
		fmt.Printf("-------------------------\n")
	}
}
