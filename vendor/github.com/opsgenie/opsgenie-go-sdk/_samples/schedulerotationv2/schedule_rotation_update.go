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

	scheduleCli, _ := cli.ScheduleRotationV2()

	identifier := &schedulev2.ScheduleIdentifier{
		Name:"Integration-schedule",
	}
	noneParticipant, err := schedulev2.NewParticipant(schedulev2.NoneParticipant, "","","")

	startHour := 3
	endHour := 15
	startMin := 5
	endMin := 30

	timeRestriction := schedulev2.TimeRestriction{
		Type: schedulev2.DayRestrictionType,
		Restriction: schedulev2.Restriction{
			StartHour: startHour,
			EndHour:   endHour,
			StartMin:  startMin,
			EndMin:    endMin,

		},
	}

	response, err := scheduleCli.Update(schedulev2.UpdateScheduleRotationRequest{
		ScheduleIdentifier: 		identifier,
		Name: 						"RotationScheduleName",
		Length:						2,
		TimeRestriction:			timeRestriction,
		Type:						schedulev2.DailyRotation,
		Participants:				[]schedulev2.Participant{
											noneParticipant,
									},
		StartDate:			"2018-05-12T09:00Z",
		EndDate:			"2018-05-12T15:00Z",
		ID:					"example-id",
	})

	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Printf(
			"ID: %s, Name: %s, Enabled: %t\n",
			response.Schedule.ID,
			response.Schedule.Name,
			response.Schedule.Enabled,
		)
	}
}
