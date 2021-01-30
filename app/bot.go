// Package gotwitchbot contains a complete Twitch.tv bot, including IRC connection.
package main

import (
	"database/sql"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/gempir/go-twitch-irc/v2"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

const (
	oauthForm = "oauth:"
	// First group is command, second group is optional permission, third group is options
	// Add username possibility to the permission category
	commandRegex = "^!(?P<trigger>\\S+) ?(?P<permission>\\+[emb])? ?(?P<options>.*)"
)

var (
	username    string
	oauth       string
	targets     []string
	commandList [][2]string
	channels    map[string]broadcaster
)

var CLIENT *twitch.Client
var RE *regexp.Regexp

type broadcaster struct {
	name      string
	database  *sql.DB
	commands  []string
	connected bool
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

/* General Twitch */

func OauthCheck() {
	zap.S().Debug("Checking OAuth Format")
	if oauth[:6] != oauthForm {
		zap.S().Debug("Fixing OAuth Format")
		oauth = oauthForm + oauth
	}
}

func Disconnectedchannel(ch broadcaster) {
	ch.connected = false
}

func ConnectedChannel(ch broadcaster) {
	ch.connected = true
}

/* Formatting */

func FormatResponse(payload string, message twitch.PrivateMessage) string {
	//Username formatting:: {user} - grabs the username of the user
	var user string
	if message.User.DisplayName != "" {
		user = message.User.DisplayName
	} else {
		user = message.User.Name
	}
	formatted := strings.ReplaceAll(payload, "{user}", user)
	//Target Formatting:: {target} - grabs the first word after the command
	match := RE.FindStringSubmatch(message.Message)
	if len(match) != 0 {
		message := match[3]
		target := strings.SplitAfterN(message, " ", 2)
		formatted = strings.ReplaceAll(formatted, "{target}", target[0])
	}

	//Multi-target formatting:: {target1}, {target2}, ... {targetn} or {1}, {2}, ..., {n}

	//Discord Formatting:: {discord}

	//Twitch Link Formatting:: {twitch} or {streamer}

	//Timer repeat:: {xs} - always at the end of the call, repeats the command after x seconds.
	return formatted
}

/* GoRoutines - Subprocesses */

func syncCommandList(ch broadcaster) {
	for ch.connected {
		time.Sleep(5 * time.Minute)
		ch.commands = GetCommands(ch.database)
	}
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
	RE = regexp.MustCompile(commandRegex)

	zap.S().Infof("Connecting Twitch Client: %v", username)
	CLIENT = twitch.NewClient(username, oauth)

	zap.S().Info("Prepare channels")
	for _, channelName := range targets {
		channelName = strings.ToLower(channelName)
		zap.S().Infof("Join channel %v", channelName)
		CLIENT.Join(channelName)

		zap.S().Debugf("##USERLIST FOR %v##\n", channelName)
		userlist, err := CLIENT.Userlist(channelName)
		if err != nil {
			zap.S().Errorf("Encountered error listing users: %v", err)
			zap.S().Infof("Skipping to next channel")
			continue
		}
		zap.S().Debugf("Users: %v\n", userlist)

		DB := ChannelDBConnect(channelName)
		comms := GetCommands(DB)
		bc := broadcaster{name: channelName, database: DB, commands: comms, connected: true}
		go syncCommandList(bc)
		channels[channelName] = bc
	}

	CLIENT.OnPrivateMessage(func(message twitch.PrivateMessage) {
		//zap.S().Debugf("%v - %v: %v\n", message.Channel, message.User.DisplayName, message.Message)
		if RE.MatchString(message.Message) {
			zap.S().Debugf("##Possible Command detected in %v!##", message.Channel)
			target := message.Channel
			commandMessage := ProcessChannelCommand(message, channels[target])
			if commandMessage != "" {
				CLIENT.Say(target, commandMessage)
			}
		}
	})

	CLIENT.OnWhisperMessage(func(message twitch.WhisperMessage) {
		zap.S().Debugf("Whisper received from %v", message.User)
		zap.S().Debugf("%v: %v\n", message.User.DisplayName, message.Message)
		if RE.MatchString(message.Message) {
			zap.S().Debugf("Whisper Command Received From: %v, Content: %v", message.User, message.Message)
			resultMessage := ProcessWhisperCommand(message)
			if resultMessage != "" {
				CLIENT.Whisper(message.User.Name, resultMessage)
			}
		}
	})

	err := CLIENT.Connect()
	if err != nil {
		zap.S().Errorf("Error connecting twitch client: %v", err)
		panic(err)
	}
}
