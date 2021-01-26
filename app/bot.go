// Package gotwitchbot contains a complete Twitch.tv bot, including IRC connection.
package main

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gempir/go-twitch-irc/v2"
	_ "github.com/mattn/go-sqlite3"
)

const (
	oauthForm = "oauth:"
	// First group is command, second group is optional permission, third group is options
	commandRegex = "^!(?P<trigger>\\S+) ?(?P<permission>\\+[emb])? ?(?P<options>.*)"
)

var (
	username    string
	oauth       string
	targets     []string
	commandList [][2]string
	channels    map[string]broadcaster
)

type broadcaster struct {
	name     string
	database *sql.DB
}

/* General AWS */

func handleAWSError(err error) {
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case rds.ErrCodeDBInstanceAlreadyExistsFault:
			fmt.Println(rds.ErrCodeDBInstanceAlreadyExistsFault, aerr.Error())
		case rds.ErrCodeInsufficientDBInstanceCapacityFault:
			fmt.Println(rds.ErrCodeInsufficientDBInstanceCapacityFault, aerr.Error())
		case rds.ErrCodeDBParameterGroupNotFoundFault:
			fmt.Println(rds.ErrCodeDBParameterGroupNotFoundFault, aerr.Error())
		case rds.ErrCodeDBSecurityGroupNotFoundFault:
			fmt.Println(rds.ErrCodeDBSecurityGroupNotFoundFault, aerr.Error())
		case rds.ErrCodeInstanceQuotaExceededFault:
			fmt.Println(rds.ErrCodeInstanceQuotaExceededFault, aerr.Error())
		case rds.ErrCodeStorageQuotaExceededFault:
			fmt.Println(rds.ErrCodeStorageQuotaExceededFault, aerr.Error())
		case rds.ErrCodeDBSubnetGroupNotFoundFault:
			fmt.Println(rds.ErrCodeDBSubnetGroupNotFoundFault, aerr.Error())
		case rds.ErrCodeDBSubnetGroupDoesNotCoverEnoughAZs:
			fmt.Println(rds.ErrCodeDBSubnetGroupDoesNotCoverEnoughAZs, aerr.Error())
		case rds.ErrCodeInvalidDBClusterStateFault:
			fmt.Println(rds.ErrCodeInvalidDBClusterStateFault, aerr.Error())
		case rds.ErrCodeInvalidSubnet:
			fmt.Println(rds.ErrCodeInvalidSubnet, aerr.Error())
		case rds.ErrCodeInvalidVPCNetworkStateFault:
			fmt.Println(rds.ErrCodeInvalidVPCNetworkStateFault, aerr.Error())
		case rds.ErrCodeProvisionedIopsNotAvailableInAZFault:
			fmt.Println(rds.ErrCodeProvisionedIopsNotAvailableInAZFault, aerr.Error())
		case rds.ErrCodeOptionGroupNotFoundFault:
			fmt.Println(rds.ErrCodeOptionGroupNotFoundFault, aerr.Error())
		case rds.ErrCodeDBClusterNotFoundFault:
			fmt.Println(rds.ErrCodeDBClusterNotFoundFault, aerr.Error())
		case rds.ErrCodeStorageTypeNotSupportedFault:
			fmt.Println(rds.ErrCodeStorageTypeNotSupportedFault, aerr.Error())
		case rds.ErrCodeAuthorizationNotFoundFault:
			fmt.Println(rds.ErrCodeAuthorizationNotFoundFault, aerr.Error())
		case rds.ErrCodeKMSKeyNotAccessibleFault:
			fmt.Println(rds.ErrCodeKMSKeyNotAccessibleFault, aerr.Error())
		case rds.ErrCodeDomainNotFoundFault:
			fmt.Println(rds.ErrCodeDomainNotFoundFault, aerr.Error())
		case rds.ErrCodeBackupPolicyNotFoundFault:
			fmt.Println(rds.ErrCodeBackupPolicyNotFoundFault, aerr.Error())
		default:
			fmt.Println(aerr.Error())
		}
	} else {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
	}
}

func OauthCheck() {
	if oauth[:6] != oauthForm {
		oauth = oauthForm + oauth
	}
}

/* Formatting */

func FormatResponse(payload string, message twitch.PrivateMessage) string {
	formatted := strings.ReplaceAll(payload, "{user}", message.User.DisplayName)
	formatted = strings.ReplaceAll(payload, "{target}", "{target}")

	return formatted
}

/* Run */

func main() {
	botDB := BotDBPrepare()
	BotDBMainTablesPrepare(botDB)
	BotDBBroadcasterAdd("hikthur", botDB)

	region := os.Getenv("AWS_REGION")

	targets := strings.Split(BotDBBroadcasterList(botDB), ";")
	username = getAWSSecret("bot-username", region)
	oauth = getAWSSecret("bot-oauth", region)

	OauthCheck()
	channels = make(map[string]broadcaster)

	// Define a regex object
	re := regexp.MustCompile(commandRegex)

	client := twitch.NewClient(username, oauth)

	for _, channelName := range targets {
		channelName = strings.ToLower(channelName)
		client.Join(channelName)
		fmt.Printf("##USERLIST FOR %v##\n", channelName)
		userlist, err := client.Userlist(channelName)
		if err != nil {
			fmt.Printf("Encountered error listing users: %v", err)
		}
		fmt.Printf("Users: %v\n", userlist)
		DB := ChannelDBPrepare(botDB, channelName)
		bc := broadcaster{name: channelName, database: DB}
		channels[channelName] = bc
	}

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		fmt.Printf("%v - %v: %v\n", message.Channel, message.User.DisplayName, message.Message)
		if re.MatchString(message.Message) {
			fmt.Println("##Possible Command detected!##")
			target := message.Channel
			command := DoCommand(message, channels[target], re)
			client.Say(target, command)
		}
	})

	err := client.Connect()
	if err != nil {
		panic(err)
	}
}
