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
	req := alerts.CreateAlertRequest{Message: samples.RandStringWithPrefix("Test", 8), Note: "Created for testing purposes", User: constants.User}
	response, alertErr := alertCli.Create(req)

	if alertErr != nil {
		panic(alertErr)
	}

	fmt.Printf("message: %s\n", response.Message)
	fmt.Printf("alert id: %s\n", response.AlertID)
	fmt.Printf("status: %s\n", response.Status)
	fmt.Printf("code: %d\n", response.Code)

	// close the alert
	getreq := alerts.GetAlertRequest{ID: response.AlertID}
	getresponse, alertErr := alertCli.Get(getreq)
	if alertErr != nil {
		panic(alertErr)
	}

	fmt.Printf("tags: %v\n", getresponse.Tags)
	fmt.Printf("count: %d\n", getresponse.Count)
	fmt.Printf("teams: %v\n", getresponse.Teams)
	fmt.Printf("recipients: %v\n", getresponse.Recipients)
	fmt.Printf("tiny id: %s\n", getresponse.TinyID)
	fmt.Printf("alias: %s\n", getresponse.Alias)
	fmt.Printf("entity: %s\n", getresponse.Entity)
	fmt.Printf("id: %s\n", getresponse.ID)
	fmt.Printf("updated at: %d\n", getresponse.UpdatedAt)
	fmt.Printf("message: %s\n", getresponse.Message)
	fmt.Printf("details: %v\n", getresponse.Details)
	fmt.Printf("source: %s\n", getresponse.Source)
	fmt.Printf("description: %s\n", getresponse.Description)
	fmt.Printf("created at: %d\n", getresponse.CreatedAt)
	fmt.Printf("is seen?: %t\n", getresponse.IsSeen)
	fmt.Printf("acknowledged?: %t\n", getresponse.Acknowledged)
	fmt.Printf("owner: %s\n", getresponse.Owner)
	fmt.Printf("actions: %s\n", getresponse.Actions)
	fmt.Printf("system data: %v\n", getresponse.SystemData)
}
