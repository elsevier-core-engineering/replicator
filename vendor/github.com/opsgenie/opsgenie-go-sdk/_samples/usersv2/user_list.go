package main

import (
	"fmt"

	"github.com/opsgenie/opsgenie-go-sdk/_samples/constants"
	ogcli "github.com/opsgenie/opsgenie-go-sdk/client"
	"github.com/opsgenie/opsgenie-go-sdk/userv2"
)

func main() {
	cli := new(ogcli.OpsGenieClient)
	cli.SetAPIKey(constants.APIKey)

	userCli, _ := cli.UserV2()

	request := userv2.ListUsersRequest{
		Limit:  10,
		Offset: 0,
		Sort:   userv2.UsernameSortField,
		Order:  userv2.AscSortType,
	}

	response, err := userCli.List(request)
	if err != nil {
		fmt.Println(err)
	} else {
		for _, user := range response.Users {
			fmt.Println(user)
		}
	}
}
