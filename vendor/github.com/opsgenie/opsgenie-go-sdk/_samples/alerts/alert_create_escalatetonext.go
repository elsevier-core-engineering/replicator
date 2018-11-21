package main


import (
	"fmt"

	"github.com/opsgenie/opsgenie-go-sdk/alerts"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	samples "github.com/opsgenie/opsgenie-go-sdk/_samples"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main(){

	cli := new (ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	alertCli, alertErr := cli.Alert()
	if alertErr != nil{
		panic(alertErr)
	}

	//create the alert
	createReq := alerts.CreateAlertRequest{Message: samples.RandStringWithPrefix("Test", 8)}
	createResp, createErr := alertCli.Create(createReq)
	if createErr != nil{
		panic(createErr)
	}

	fmt.Printf("message: %s\n", createResp.Message)
	fmt.Printf("alertID: %s\n", createResp.AlertID)
	fmt.Printf("status: %s\n", createResp.Status)
	fmt.Printf("code: %d\n", createResp.Code)

	// escalate to next
	escalateToNextReq := alerts.EscalateToNextAlertRequest{ID: createResp.AlertID, EscalationName: constants.EscalationName}
	escalateToNextResp, escalateToNextErr := alertCli.EscalateToNext(escalateToNextReq)
	if escalateToNextErr != nil{
		panic(escalateToNextErr)
	}

	fmt.Printf("status: %s\n", escalateToNextResp.Status)
	fmt.Printf("code: %d\n", escalateToNextResp.Code)
}
