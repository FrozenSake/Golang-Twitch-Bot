// Package gotwitchbot contains a complete Twitch.tv bot, including IRC connection.
package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
	"go.uber.org/zap"
)

/* Commands */
func ProcessUserPermissions(userBadges map[string]int) string {
	var userLevel string
	if userBadges["broadcaster"] == 1 {
		zap.S().Debug("User is the broadcaster")
		userLevel = "b"
	} else if userBadges["moderator"] == 1 {
		zap.S().Debug("User is a moderator")
		userLevel = "m"
	} else {
		zap.S().Debug("User is a viewer")
		userLevel = "v"
	}
	return userLevel
}

func AuthorizeCommand(userLevel, permissionLevel string) bool {
	zap.S().Debugf("Authorizing a command")
	if permissionLevel == "b" && userLevel != "b" {
		return false
	} else if userLevel == "m" || userLevel == "b" {
		return false
	} else {
		return true
	}
}

func ProcessUserBits(userBadges map[string]int) int {
	bits := userBadges["bits"]
	return bits
}

func ProcessUserSubscription(tags map[string]string) string {
	subscriberTime := tags["badge-info"]
	return subscriberTime
}

func ProcessWhisperCommand(message twitch.WhisperMessage, re *regexp.Regexp) string {
	zap.S().Debug("Processing Whisper Command")

	var resultMessage string
	username := message.User.Name
	submatch := re.FindStringSubmatch(message.Message)
	trigger := submatch[1]
	options := submatch[3]

	switch trigger {
	case "joinChannel":
		zap.S().Debug("Join Channel Command Called")
		BotDBBroadcasterAdd(username)
		resultMessage = "Thank you %s for the join request, I've sent it to Hikthur for authorization"
	case "authorizeJoin":
		if strings.ToLower(username) != "hikthur" {
			resultMessage = "I'm sorry, only Hikthur can authorize new channels."
		} else {
			BroadcasterAuthorize(options)
			resultMessage = fmt.Sprintf("Authorizing %s as a broadcaster.", options)
		}
	default:
		zap.S().Debug("Not a bot level command, passing back no command message.")
		resultMessage = "That is not a command I understand, please contact Hikthur with what you're trying to do."
	}
	return resultMessage
}

func ProcessChannelCommand(message twitch.PrivateMessage, ch broadcaster, re *regexp.Regexp) string {
	zap.S().Debugf("Executing a command")

	///// REWORK TO INCLUDE command permission options structure.
	submatch := re.FindStringSubmatch(message.Message)
	trigger := submatch[1]
	level := submatch[2]
	options := submatch[3]
	var result string

	userPermissionLevel := ProcessUserPermissions(message.User.Badges) //Pre-processed by twitchirc
	var requiredPermission string
	switch trigger {
	case "addcommand":
		requiredPermission = "m"
		if !AuthorizeCommand(userPermissionLevel, requiredPermission) {
			result = ""
		} else {
			submatch = re.FindStringSubmatch(options)
			newTrigger := submatch[1]
			newOptions := submatch[2]
			result = CommandDBInsert(newTrigger, newOptions, level, ch.database, 0)
		}
	case "removecommand":
		requiredPermission = "m"
		if !AuthorizeCommand(userPermissionLevel, requiredPermission) {
			result = ""
		} else {
			submatch = re.FindStringSubmatch(options)
			deleteTrigger := submatch[1]
			result = CommandDBRemove(deleteTrigger, ch.database)
		}
	case "connectiontest":
		requiredPermission = "m"
		if !AuthorizeCommand(userPermissionLevel, requiredPermission) {
			result = ""
		} else {
			result = "The bot has succesfully latched on to this channel."
		}
	default:
		result, requiredPermission := CommandDBSelect(trigger, ch.database)
		if result == "" {
			result = "No " + trigger + " command."
		}
		if !AuthorizeCommand(userPermissionLevel, requiredPermission) {
			result = "Sorry, you're not authorized to use this command {user}."
		}
	}

	result = FormatResponse(result, message)

	return result
}
