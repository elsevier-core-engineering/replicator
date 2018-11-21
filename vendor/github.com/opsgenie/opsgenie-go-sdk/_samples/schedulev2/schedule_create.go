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

	scheduleCli, _ := cli.ScheduleV2()


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

	ownerTeam := schedulev2.OwnerTeam{
			Name: "Integrations",
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
	escalationParticipantWithName, err := schedulev2.NewParticipant(schedulev2.EscalationParticipant, "","","")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		participants = append(participants, escalationParticipantWithName)
	}
	teamParticipantWithID, err := schedulev2.NewParticipant(schedulev2.TeamParticipant,"927b4905-4be2-43dd-808c-0a3a33113bca","","")
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

	response, err := scheduleCli.Create(schedulev2.CreateScheduleRequest{
		Name:            "Integration-schedule",
		Description:	 "This schedule created for test purpose.",
		Timezone:		 "Europe/Kirov",
		OwnerTeam:		 ownerTeam,
		Rotations:		 rotations,
		Enabled:		 true,
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
