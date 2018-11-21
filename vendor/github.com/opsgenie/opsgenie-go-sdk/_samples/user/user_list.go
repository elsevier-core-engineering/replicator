package main

import (
	"fmt"

	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/opsgenie/opsgenie-go-sdk/user"
	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	userCli, cliErr := cli.User()

	if cliErr != nil {
		panic(cliErr)
	}

	req := user.ListUsersRequest{}
	response, userErr := userCli.List(req)

	if userErr != nil {
		panic(userErr)
	}

	for _, user := range response.Users {
		fmt.Printf("Id: %s\n", user.Id)
		fmt.Printf("Username: %s\n", user.Username)
		fmt.Printf("Fullname: %s\n", user.Fullname)
		fmt.Printf("Timezone: %s\n", user.Timezone)
		fmt.Printf("Locale: %s\n", user.Locale)
		fmt.Printf("State: %s\n", user.State)
		fmt.Printf("Escalations: %v\n", user.Escalations)
		fmt.Printf("Schedules: %v\n", user.Schedules)
		fmt.Printf("Role: %v\n", user.Role)
		fmt.Printf("Groups: %v\n", user.Groups)
		fmt.Printf("Contacts: %v\n", user.Contacts)
		fmt.Printf("\n")
	}
}
