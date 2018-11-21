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

	identifier := &schedulev2.Identifier{
		Name:"Integration-schedule",
	}
	noneParticipant, err := schedulev2.NewParticipant(schedulev2.NoneParticipant, "","","")
	userParticipant, err := schedulev2.NewParticipant(schedulev2.UserParticipant, "","","user@company.com")

	restrictions := []schedulev2.Restriction{
		{
			StartDay:	schedulev2.Monday,
			StartHour:8,
			StartMin:7,
			EndDay:		schedulev2.Friday,
			EndHour:18,
			EndMin:30,
		},
		{
			StartDay:schedulev2.Wednesday,
			StartHour:8,
			StartMin:8,
			EndDay:schedulev2.Thursday,
			EndHour:18,
			EndMin:30,
		},
	}

	rotations := []schedulev2.Rotation{
		{
			Name:	"rotation1",
			StartDate:	"2018-01-15T08:00:00+02:00",
			EndDate:	"2018-02-15T08:00:00+02:00",
			Type:		schedulev2.DailyRotation,
			Participants:	[]schedulev2.Participant{
				noneParticipant,
				userParticipant,
				},
			TimeRestriction:schedulev2.TimeRestriction{
				Type:schedulev2.WeekDayRestrictionType,
				Restrictions:restrictions,
			},
			Length:2,
		},
	}

	response, err := scheduleCli.Update(schedulev2.UpdateScheduleRequest{
		Identifier: identifier,
		Name: "DisabledScheduleName",
		Enabled:		 false,
		Rotations:		 rotations,
		OwnerTeam:					schedulev2.OwnerTeam{
										Name:"Integrations",
									},
		Description:				"this description been written to make test",

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
