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

func AuthorizeCommand(userLevel, username, permissionLevel string) bool {
	zap.S().Debugf("Authorizing a command")
	if username == permissionLevel {
		zap.S().Debugf("User is the explicit allow to perform this command.")
		return true
	} else if userLevel == "b" {
		zap.S().Debugf("The broadcaster can execute any command.")
		return true
	} else if permissionLevel == "m" && userLevel != "m" {
		zap.S().Debugf("User is not authorized for moderator level commands")
		return false
	} else if permissionLevel == "" {
		zap.S().Debugf("This command is available to all users.")
		return true
	} else {
		zap.S().Debugf("Default deny")
		return false
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
	trigger := strings.ToLower(submatch[1])
	options := submatch[3]

	switch trigger {
	case "joinchannel":
		zap.S().Debug("Join Channel Command Called")
		BotDBBroadcasterAdd(username)
		CLIENT.Whisper("Hikthur", fmt.Sprintf("%s would like me to join their channel, thoughts? Use !authorizejoin to approve.", username))
		resultMessage = fmt.Sprintf("Thank you %s for the join request, I've sent it to Hikthur for authorization", message.User.Name)
	case "authorizejoin":
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
	trigger := strings.ToLower(submatch[1])
	level := strings.ToLower(submatch[2])
	options := submatch[3]
	var result string

	userName := message.User.Name
	userPermissionLevel := ProcessUserPermissions(message.User.Badges) //Pre-processed by twitchirc
	var requiredPermission string
	switch trigger {
	case "addcommand":
		requiredPermission = "m"
		if !AuthorizeCommand(userPermissionLevel, userName, requiredPermission) {
			result = ""
		} else {
			submatch = re.FindStringSubmatch(options)
			if len(submatch) == 0 {
				result = "I'm sorry, I can't add that command for some reason."
			} else {
				result = CommandDBInsert(submatch[1], submatch[2], level, 0, ch.database)
			}
		}
	case "removecommand":
		requiredPermission = "m"
		if !AuthorizeCommand(userPermissionLevel, userName, requiredPermission) {
			result = ""
		} else {
			submatch = re.FindStringSubmatch(options)
			deleteTrigger := submatch[1]
			result = CommandDBRemove(deleteTrigger, ch.database)
		}
	case "connectiontest":
		requiredPermission = "m"
		if !AuthorizeCommand(userPermissionLevel, userName, requiredPermission) {
			result = ""
		} else {
			result = "The bot has succesfully latched on to this channel."
		}
	default:
		res, requiredPermission := CommandDBSelect(trigger, ch.database)
		if res == "" {
			result = "No " + trigger + " command."
		} else if !AuthorizeCommand(userPermissionLevel, userName, requiredPermission) {
			result = "Sorry, you're not authorized to use this command {user}."
		} else {
			result = res
		}
	}

	result = FormatResponse(result, message)

	return result
}
