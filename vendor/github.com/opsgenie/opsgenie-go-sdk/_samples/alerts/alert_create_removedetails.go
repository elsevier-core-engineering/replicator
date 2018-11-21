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

	//create the alert with details
	createReq := alerts.CreateAlertRequest{Message: samples.RandStringWithPrefix("Test", 8), Details: map[string]string{"detail1" : "detail1", "detail2" : "detail2"}}
	createResp, createErr := alertCli.Create(createReq)
	if createErr != nil{
		panic(createErr)
	}

	fmt.Printf("message: %s\n", createResp.Message)
	fmt.Printf("alertID: %s\n", createResp.AlertID)
	fmt.Printf("status: %s\n", createResp.Status)
	fmt.Printf("code: %d\n", createResp.Code)

	// remove details from the alert
	removeDetailsReq := alerts.RemoveDetailsAlertRequest{ID: createResp.AlertID, Keys: []string{"detail1", "detail2"}}
	removeDetailsResp, removeDetailsErr := alertCli.RemoveDetails(removeDetailsReq)
	if removeDetailsErr != nil{
		panic(removeDetailsErr)
	}

	fmt.Printf("status: %s\n", removeDetailsResp.Status)
	fmt.Printf("code: %d\n", removeDetailsResp.Code)
}
