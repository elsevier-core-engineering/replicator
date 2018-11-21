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

	//req := sch.CreateScheduleRequest{Name: "", Timezone: "", Enabled: &enabled, Rotations: rotations}

	createReq := sch.CreateScheduleRequest{ Name: "test", Enabled: &enabled, Rotations: rotations_}
	createResp, createErr := scheduleCli.Create(createReq)
	if createErr != nil {
		panic(createErr)
	}

	timelineReq := sch.GetTimelineScheduleRequest{ Id: createResp.Id, Name: "test"}
	timelineResp, timelineErr := scheduleCli.GetTimeline(timelineReq)
	if timelineErr != nil {
		panic(timelineErr)
	}

	schedule := timelineResp.Schedule
	fmt.Printf("timezone: %s\n", schedule.Timezone)
	fmt.Printf("name: %s\n", schedule.Name)
	fmt.Printf("id: %s\n", schedule.Id)
	fmt.Printf("team: %s\n", schedule.Team)
	fmt.Printf("enabled: %t\n", schedule.Enabled)

	timeline := timelineResp.Timeline
	fmt.Printf("starttime: %d\n", timeline.StartTime)
	fmt.Printf("endtime: %d\n", timeline.EndTime)
	rotations := timeline.FinalSchedule.Rotations
	for _, rot := range rotations{
		fmt.Printf("rotations:\n")
		fmt.Printf("rotName: %s\n", rot.Name)
		fmt.Printf("rotId: %s\n", rot.Id)
		fmt.Printf("rotOrder: %f\n", rot.Order)
		periods := rot.Periods
		for _, period := range periods{
			fmt.Printf("periods:\n")
			fmt.Printf("periodStart: %d\n", period.StartTime)
			fmt.Printf("periodEnd: %d\n", period.EndTime)
			fmt.Printf("periodType: %s\n", period.Type)
			recipients := period.Recipients
			for _, rec := range recipients{
				fmt.Printf("recipients:\n")
				fmt.Printf("recDisplayName: %s\n", rec.DisplayName)
				fmt.Printf("recName: %s\n", rec.Name)
				fmt.Printf("recId: %s\n", rec.Id)
				fmt.Printf("recType: %s\n", rec.Type)
			}
			fromUsers := period.FromUsers
			for _, user := range fromUsers{
				fmt.Printf("fromUsers:\n")
				fmt.Printf("fromUserDisplayName: %s\n", user.DisplayName)
				fmt.Printf("fromUserName: %s\n", user.Name)
				fmt.Printf("fromUserId: %s\n", user.Id)
				fmt.Printf("fromUserType: %s\n", user.Type)
			}
		}
	}
}
