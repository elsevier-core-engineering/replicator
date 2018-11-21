package main

import(
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"fmt"
	"github.com/opsgenie/opsgenie-go-sdk/schedulev2"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	scheduleCli, _ := cli.ScheduleOverrideV2()

	identifier := &schedulev2.ScheduleIdentifier{
		Name:"Integration-schedule",
	}
	noneParticipant, err := schedulev2.NewParticipant(schedulev2.NoneParticipant, "","","")

	response, err := scheduleCli.Update(schedulev2.UpdateScheduleOverrideRequest{
		ScheduleIdentifier: identifier,
		Alias:				"Override Alias",
		StartDate:			"2018-05-12T09:00Z",
		EndDate:			"2018-05-12T15:00Z",
		User:				schedulev2.User{
								Username: 	"user@company.com",
								Type:		schedulev2.UserUserType,
							},
		Rotations:[]schedulev2.Rotation{
			{
				Name:			"test-rotation",
				TimeRestriction:schedulev2.TimeRestriction{
					Type:schedulev2.DayRestrictionType,
					Restriction:schedulev2.Restriction{
						EndMin:1,
						StartMin:0,
						StartHour:1,
						EndHour:2,
					},
				},
				Participants:[]schedulev2.Participant{
					noneParticipant,
				},
			},
		},
	})

	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Printf(
			"Alias: %s, StartDate: %s, EndDate: %t\n",
			response.ScheduleOverride.Alias,
			response.ScheduleOverride.StartDate,
			response.ScheduleOverride.EndDate,
		)
	}
}
