package main

import (
	"github.com/opsgenie/opsgenie-go-sdk/alertsv2"
	"fmt"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	alertCli, _ := cli.AlertV2()

	teams := []alertsv2.TeamRecipient{
		&alertsv2.Team{Name: "teamId"},
		&alertsv2.Team{ID: "teamId"},
	}

	visibleTo := [] alertsv2.Recipient{
		&alertsv2.Team{ID: "teamId"},
		&alertsv2.Team{Name: "teamName"},
		&alertsv2.User{ID: "userId"},
		&alertsv2.User{Username: "user@opsgenie.com"},
	}

	request :=
		alertsv2.CreateAlertRequest{
			Message:     "message",
			Alias:       "alias",
			Description: "alert description",
			Teams:       teams,
			VisibleTo:   visibleTo,
			Actions:     []string{"action1", "action2"},
			Tags:        []string{"tag1", "tag2"},
			Details: map[string]string{
				"key":  "value",
				"key2": "value2",
			},
			Entity:   "entity",
			Source:   "source",
			Priority: alertsv2.P1,
			User:     "user@opsgenie.com",
			Note:     "alert note",
		}

	response, err := alertCli.Create(request)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Create request ID: " + response.RequestID)
	}
}
