// Package gotwitchbot contains a complete Twitch.tv bot, including IRC connection.
package main

import (
	"database/sql"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gempir/go-twitch-irc/v2"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
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
			zap.S().Errorf(rds.ErrCodeDBInstanceAlreadyExistsFault, aerr.Error())
		case rds.ErrCodeInsufficientDBInstanceCapacityFault:
			zap.S().Errorf(rds.ErrCodeInsufficientDBInstanceCapacityFault, aerr.Error())
		case rds.ErrCodeDBParameterGroupNotFoundFault:
			zap.S().Errorf(rds.ErrCodeDBParameterGroupNotFoundFault, aerr.Error())
		case rds.ErrCodeDBSecurityGroupNotFoundFault:
			zap.S().Errorf(rds.ErrCodeDBSecurityGroupNotFoundFault, aerr.Error())
		case rds.ErrCodeInstanceQuotaExceededFault:
			zap.S().Errorf(rds.ErrCodeInstanceQuotaExceededFault, aerr.Error())
		case rds.ErrCodeStorageQuotaExceededFault:
			zap.S().Errorf(rds.ErrCodeStorageQuotaExceededFault, aerr.Error())
		case rds.ErrCodeDBSubnetGroupNotFoundFault:
			zap.S().Errorf(rds.ErrCodeDBSubnetGroupNotFoundFault, aerr.Error())
		case rds.ErrCodeDBSubnetGroupDoesNotCoverEnoughAZs:
			zap.S().Errorf(rds.ErrCodeDBSubnetGroupDoesNotCoverEnoughAZs, aerr.Error())
		case rds.ErrCodeInvalidDBClusterStateFault:
			zap.S().Errorf(rds.ErrCodeInvalidDBClusterStateFault, aerr.Error())
		case rds.ErrCodeInvalidSubnet:
			zap.S().Errorf(rds.ErrCodeInvalidSubnet, aerr.Error())
		case rds.ErrCodeInvalidVPCNetworkStateFault:
			zap.S().Errorf(rds.ErrCodeInvalidVPCNetworkStateFault, aerr.Error())
		case rds.ErrCodeProvisionedIopsNotAvailableInAZFault:
			zap.S().Errorf(rds.ErrCodeProvisionedIopsNotAvailableInAZFault, aerr.Error())
		case rds.ErrCodeOptionGroupNotFoundFault:
			zap.S().Errorf(rds.ErrCodeOptionGroupNotFoundFault, aerr.Error())
		case rds.ErrCodeDBClusterNotFoundFault:
			zap.S().Errorf(rds.ErrCodeDBClusterNotFoundFault, aerr.Error())
		case rds.ErrCodeStorageTypeNotSupportedFault:
			zap.S().Errorf(rds.ErrCodeStorageTypeNotSupportedFault, aerr.Error())
		case rds.ErrCodeAuthorizationNotFoundFault:
			zap.S().Errorf(rds.ErrCodeAuthorizationNotFoundFault, aerr.Error())
		case rds.ErrCodeKMSKeyNotAccessibleFault:
			zap.S().Errorf(rds.ErrCodeKMSKeyNotAccessibleFault, aerr.Error())
		case rds.ErrCodeDomainNotFoundFault:
			zap.S().Errorf(rds.ErrCodeDomainNotFoundFault, aerr.Error())
		case rds.ErrCodeBackupPolicyNotFoundFault:
			zap.S().Errorf(rds.ErrCodeBackupPolicyNotFoundFault, aerr.Error())
		default:
			zap.S().Errorf(aerr.Error())
		}
	} else {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		zap.S().Errorf(err.Error())
	}
}

func OauthCheck() {
	zap.S().Debug("Checking OAuth Format")
	if oauth[:6] != oauthForm {
		zap.S().Debug("Fixing OAuth Format")
		oauth = oauthForm + oauth
	}
}

/* Formatting */

func FormatResponse(payload string, message twitch.PrivateMessage) string {
	var user string
	if message.User.Name == "" && message.User.DisplayName != "" {
		user = message.User.DisplayName
	} else {
		user = message.User.Name
	}
	formatted := strings.ReplaceAll(payload, "{user}", user)

	return formatted
}

/* Run */

func main() {
	sugar, _ := zap.NewDevelopment()
	defer sugar.Sync()

	zap.ReplaceGlobals(sugar)

	zap.S().Info("Twitch Chatbot Starting up.")

	zap.S().Info("Begin BotDB Preparation Stack.")
	BotDBPrepare()
	BotDBMainTablesPrepare()
	zap.S().Info("BotDB Preparation Stack Complete.")

	zap.S().Debug("Setting Environment Variables")
	targets := strings.Split(BotDBBroadcasterList(), ";")
	region := os.Getenv("AWS_REGION")
	username = getAWSSecret("bot-username", region)
	oauth = getAWSSecret("bot-oauth", region)

	OauthCheck()
	channels = make(map[string]broadcaster)

	// Define a regex object
	re := regexp.MustCompile(commandRegex)

	zap.S().Infof("Connecting Twitch Client: %v", username)
	client := twitch.NewClient(username, oauth)

	zap.S().Info("Prepare channels")
	for _, channelName := range targets {
		channelName = strings.ToLower(channelName)
		zap.S().Infof("Join channel %v", channelName)
		client.Join(channelName)

		zap.S().Debugf("##USERLIST FOR %v##\n", channelName)
		userlist, err := client.Userlist(channelName)
		if err != nil {
			zap.S().Errorf("Encountered error listing users: %v", err)
			zap.S().Infof("Skipping to next channel")
			continue
		}
		zap.S().Debugf("Users: %v\n", userlist)

		DB := ChannelDBConnect(channelName)
		bc := broadcaster{name: channelName, database: DB}
		channels[channelName] = bc
	}

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		//zap.S().Debugf("%v - %v: %v\n", message.Channel, message.User.DisplayName, message.Message)
		if re.MatchString(message.Message) {
			zap.S().Debugf("##Possible Command detected in %v!##", message.Channel)
			target := message.Channel
			command := ProcessChannelCommand(message, channels[target], re)
			if command != "" {
				client.Say(target, command)
			}
		}
	})

	client.OnWhisperMessage(func(message twitch.WhisperMessage) {
		zap.S().Debugf("Whisper received from %v", message.User)
		if re.MatchString(message.Message) {
			zap.S().Debugf("Whisper Command Received From: %v, Content: %v", message.User, message.Message)
			ProcessWhisperCommand(message, re)
		}
	})

	err := client.Connect()
	if err != nil {
		zap.S().Errorf("Error connecting twitch client: %v", err)
		panic(err)
	}
}
