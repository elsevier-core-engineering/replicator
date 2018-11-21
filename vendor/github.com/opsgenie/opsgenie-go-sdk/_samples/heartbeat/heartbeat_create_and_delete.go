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

	fmt.Printf("Heartbeat created\n")
	fmt.Printf("-----------------\n")
	fmt.Printf("name: %s\n", response.Name)
	fmt.Printf("status: %s\n", response.Status)
	fmt.Printf("code: %d\n", response.Code)

	// enable the hb
	deleteReq := hb.DeleteHeartbeatRequest{Name: response.Name}
	deleteResp, deleteErr := hbCli.Delete(deleteReq)
	if deleteErr != nil {
		panic(deleteErr)
	}

	fmt.Println()
	fmt.Printf("Heartbeat deleted\n")
	fmt.Printf("-----------------\n")
	fmt.Printf("Status: %s\n", deleteResp.Status)
	fmt.Printf("Code: %d\n", deleteResp.Code)
}
