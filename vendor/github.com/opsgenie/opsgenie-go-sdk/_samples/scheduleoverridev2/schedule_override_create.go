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


	startHour := 3
	endHour := 15
	startMin := 5
	endMin := 30

	identifier := &schedulev2.ScheduleIdentifier{
		Name:"Integration-schedule",
	}

	user := schedulev2.User{
		Type:		schedulev2.UserUserType,
		Username:	"user@company.com",
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

	rotations := []schedulev2.Rotation{
		{
			Name:					"test-rotation",
			StartDate:				"2017-01-15T08:00:00+02:00",
			Length:					2,
			Participants:			participants,
			TimeRestriction: 		timeRestriction,
			Type:					schedulev2.DailyRotation,
		},
	}

	response, err := scheduleCli.Create(schedulev2.CreateScheduleOverrideRequest{
		ScheduleIdentifier: identifier,
		Alias:				"Override Alias",
		User: 				user,
		StartDate:			"2018-05-12T09:00Z",
		EndDate:			"2018-05-12T15:00Z",
		Rotations:		 	rotations,
	})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Printf(
			"Alias: %s, StartDate: %s, EndDate: %s\n",
			response.ScheduleOverride.Alias,
			response.ScheduleOverride.StartDate,
			response.ScheduleOverride.EndDate,
		)
	}
}
