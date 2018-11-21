package main

import (
	"fmt"

	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	sch "github.com/opsgenie/opsgenie-go-sdk/schedule"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main(){
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	scheduleCli, cliErr := cli.Schedule()
	if cliErr != nil {
		panic(cliErr)
	}

	restrictions := []sch.Restriction{}
	restriction := sch.Restriction{StartDay: "monday", StartTime: "08:00", EndDay: "thursday", EndTime: "17:00"}
	restrictions = append(restrictions, restriction)

	rotations_ := []sch.Rotation{}
	rotation_ := sch.Rotation{Name: "Off Hours", StartDate: "2013-02-10 18:00", EndDate: "2013-07-10 12:00", Participants: []string{"group1"}, RotationType: "hourly", RotationLength: 8, Restrictions: restrictions}
	rotation2_ := sch.Rotation{Name: "Another Off Hours", StartDate: "2013-02-10 18:00", EndDate: "2013-07-10 12:00", Participants: []string{"group1", "fazilet@test.com"}, RotationType: "daily", RotationLength: 8, Restrictions: restrictions}
	rotations_ = append(rotations_, rotation_)
	rotations_ = append(rotations_, rotation2_)

	enabled := true

	createReq := sch.CreateScheduleRequest{ Name: "test", Enabled: &enabled, Rotations: rotations_}
	createResp, createErr := scheduleCli.Create(createReq)
	if createErr != nil {
		panic(createErr)
	}


	onCallReq := sch.WhoIsOnCallRequest{ Id: createResp.Id, Name: "test"}
	onCallResp, onCallErr := scheduleCli.WhoIsOnCall(onCallReq)
	if onCallErr != nil{
		panic(onCallErr)
	}

	fmt.Printf("id: %s\n", onCallResp.Id)
	fmt.Printf("name: %s\n", onCallResp.Name)
	fmt.Printf("type: %s\n", onCallResp.Type)
	participants := onCallResp.Participants
	for _, part := range participants{
		fmt.Printf("name: %s\n", part.Name)
		fmt.Printf("type: %s\n", part.Type)
		fmt.Printf("forwarded: %t\n", part.Forwarded)
		fmt.Printf("notifyType: %s\n", part.NotifyType)

	}
}
