package main

import (
	"fmt"

	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	sch "github.com/opsgenie/opsgenie-go-sdk/schedule"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	userCli, cliErr := cli.Schedule()

	if cliErr != nil {
		panic(cliErr)
	}

	restrictions := []sch.Restriction{}
	restriction := sch.Restriction{StartDay: "", StartTime: "", EndDay: "", EndTime: ""}
	restrictions = append(restrictions, restriction)

	rotations := []sch.Rotation{}
	rotation := sch.Rotation{Name: "", StartDate: "", EndDate: "", Participants: []string{""}, RotationType: "", RotationLength: 8, Restrictions: restrictions}
	rotations = append(rotations, rotation)

	enabled := true

	req := sch.CreateScheduleRequest{Name: "", Timezone: "", Enabled: &enabled, Rotations: rotations}
	response, userErr := userCli.Create(req)

	if userErr != nil {
		panic(userErr)
	}

	fmt.Printf("id: %s\n", response.Id)
	fmt.Printf("status: %s\n", response.Status)
	fmt.Printf("code: %d\n", response.Code)
}
