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
		Alias: "override_add",
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

	getReq := override.GetScheduleOverrideRequest{ Alias: overrideResponse.Alias, Schedule: "test"}
	getResp, getErr := overrideCli.Get(getReq)
	if getErr != nil{
		panic(getErr)
	}

	fmt.Printf("alias: %s\n", getResp.Alias)
	fmt.Printf("user: %s\n", getResp.User)
	fmt.Printf("startDate: %s\n", getResp.StartDate)
	fmt.Printf("endDate: %s\n", getResp.EndDate)
	fmt.Printf("timezone: %s\n", getResp.Timezone)
	rotationIds := getResp.RotationIds
	for _, id := range rotationIds{
		fmt.Printf("id: %s\n", id)
	}

}

