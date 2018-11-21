package main

import (
	"fmt"

	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	hb "github.com/opsgenie/opsgenie-go-sdk/heartbeat"
	samples "github.com/opsgenie/opsgenie-go-sdk/_samples"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	hbCli, cliErr := cli.Heartbeat()

	if cliErr != nil {
		panic(cliErr)
	}

	// create the hb
	enabled := true
	req := hb.AddHeartbeatRequest{
		Name: samples.RandStringWithPrefix("Test", 4),
		IntervalUnit:"minutes",
		Enabled: &enabled,
		Interval:5,
		Description:"Heartbeat description"}
	response, hbErr := hbCli.Add(req)

	if hbErr != nil {
		panic(hbErr)
	}

	fmt.Printf("Heartbeat added\n")
	fmt.Printf("---------------\n")
	fmt.Printf("name: %s\n", response.Name)
	fmt.Printf("status: %s\n", response.Status)
	fmt.Printf("code: %d\n", response.Code)

	// update the newly created heart beat, change description
	updateReq := hb.UpdateHeartbeatRequest{Name: response.Name, Description: "new description"}
	updateResp, updateErr := hbCli.Update(updateReq)

	if updateErr != nil {
		panic(updateErr)
	}

	fmt.Println()
	fmt.Printf("Heartbeat updated\n")
	fmt.Printf("-----------------\n")
	fmt.Printf("name: %s\n", updateResp.Name)
	fmt.Printf("status: %s\n", updateResp.Status)
	fmt.Printf("code: %d\n", updateResp.Code)

	getReq := hb.GetHeartbeatRequest{Name: response.Name}
	getResp, getErr := hbCli.Get(getReq)
	if getErr != nil {
		panic(getErr)
	}

	fmt.Println()
	fmt.Printf("Heartbeat details\n")
	fmt.Printf("-----------------\n")
	fmt.Printf("Name: %s\n", getResp.Name)
	fmt.Printf("Description: %s\n", getResp.Description)
}
