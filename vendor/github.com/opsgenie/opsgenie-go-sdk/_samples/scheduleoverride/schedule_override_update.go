package main

import (
	"fmt"

	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/opsgenie/opsgenie-go-sdk/scheduleoverride"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	overrideCli, cliErr := cli.ScheduleOverride()
	if cliErr != nil {
		panic(cliErr)
	}
	req := override.AddScheduleOverrideRequest{
		Alias: "override_update",
		Schedule: "test",
		User: "fazilet@test.com",
		StartDate: "2013-01-27 22:00",
		EndDate: "2018-01-28 08:00"}
	overrideResponse, overrideErr := overrideCli.Add(req)

	if overrideErr != nil {
		panic(overrideErr)
	}

	fmt.Printf("alias: %s\n", overrideResponse.Alias)
	fmt.Printf("status: %s\n", overrideResponse.Status)
	fmt.Printf("code: %d\n", overrideResponse.Code)

	updateReq := override.UpdateScheduleOverrideRequest{
		Alias: overrideResponse.Alias,
		Schedule: "test",
		User: "fazilet@test.com",
		StartDate: "2013-01-27 22:00",
		EndDate: "2019-01-28 08:00"}
	updateResp, updateErr := overrideCli.Update(updateReq)
	if updateErr != nil{
		panic(updateErr)
	}

	fmt.Printf("alias: %s\n", updateResp.Alias)
	fmt.Printf("status: %s\n", updateResp.Status)
	fmt.Printf("code: %d\n", updateResp.Code)

}

