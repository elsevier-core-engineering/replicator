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


	startHour := 3
	endHour := 15
	startMin := 5
	endMin := 30

	identifier := &schedulev2.ScheduleIdentifier{
		Name:"Integration-schedule",
	}

	timeRestriction := schedulev2.TimeRestriction{
		Type: schedulev2.DayRestrictionType,
		Restriction: schedulev2.Restriction{
			StartHour: startHour,
			EndHour:   endHour,
			StartMin:  startMin,
			EndMin:    endMin,

		},
	}

	var participants []schedulev2.Participant
	noneParticipant, err := schedulev2.NewParticipant(schedulev2.NoneParticipant, "", "","")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		participants = append(participants, noneParticipant)
	}
	userParticipantWithUsername, err := schedulev2.NewParticipant(schedulev2.UserParticipant,"","","user@company.com")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		participants = append(participants, userParticipantWithUsername)
	}
	escalationParticipantWithName, err := schedulev2.NewParticipant(schedulev2.EscalationParticipant, "","example-escalation-name","")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		participants = append(participants, escalationParticipantWithName)
	}
	teamParticipantWithID, err := schedulev2.NewParticipant(schedulev2.TeamParticipant,"example-team-id","","")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		participants = append(participants, teamParticipantWithID)
	}

	response, err := scheduleCli.Create(schedulev2.CreateScheduleRotationRequest{
		ScheduleIdentifier: identifier,
		Name:				"Rotation-test",
		Type:				schedulev2.DailyRotation,
		Participants:		participants,
		TimeRestriction:	timeRestriction,
		StartDate:			"2018-05-12T09:00Z",
		EndDate:			"2018-05-12T15:00Z",
		Length:				2,
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
