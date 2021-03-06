// Package gotwitchbot contains a complete Twitch.tv bot, including IRC connection.
package main

import (
	"fmt"
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
	} else if permissionLevel == "m" && userLevel == "m" {
		zap.S().Debugf("User is authorized for moderator level commands")
		return true
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

func ProcessWhisperCommand(message twitch.WhisperMessage) string {
	zap.S().Debug("Processing Whisper Command")

	var resultMessage string
	username := message.User.Name
	submatch := RE.FindStringSubmatch(message.Message)
	trigger := strings.ToLower(submatch[1])
	options := submatch[3]

	switch trigger {
	case "joinchannel":
		zap.S().Debug("Join Channel Command Called")
		BotDBBroadcasterAdd(username)
		CLIENT.Whisper("hikthur", fmt.Sprintf("%s would like me to join their channel, thoughts? Use !authorizejoin to approve.", username))
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

func ProcessChannelCommand(message twitch.PrivateMessage, ch broadcaster) string {
	zap.S().Debugf("Executing a command")

	///// REWORK TO INCLUDE command permission options structure.
	submatch := RE.FindStringSubmatch(message.Message)
	trigger := strings.ToLower(submatch[1])
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
			submatch = RE.FindStringSubmatch(options)
			if len(submatch) == 0 {
				result = "I'm sorry, I can't add that command for some reason."
			} else {
				newTrigger := submatch[1]
				newLevel := strings.ToLower(submatch[2])
				newPayload := submatch[3]
				zap.S().Debugf("Adding command with trigger: %v, level: %v, payload: %v", newTrigger, newLevel, newPayload)
				result = CommandDBInsert(newTrigger, newPayload, newLevel, 0, ch.database)
				if result != "I couldn't add that command due to a SQL error." {
					ch.commands = append(ch.commands, newTrigger)
				}
			}
		}
	case "removecommand":
		requiredPermission = "m"
		if !AuthorizeCommand(userPermissionLevel, userName, requiredPermission) {
			result = ""
		} else {
			submatch = RE.FindStringSubmatch(options)
			if len(submatch) == 0 {
				result = "I'm sorry, you didn't supply a command I understand."
			} else {
				deleteTrigger := submatch[1]
				result = CommandDBRemove(deleteTrigger, ch.database)
			}
		}
	case "connectiontest":
		requiredPermission = "m"
		if !AuthorizeCommand(userPermissionLevel, userName, requiredPermission) {
			result = ""
		} else {
			result = "The bot has succesfully latched on to this channel."
		}
	case "help":
		result = "This bot is being helpful!"
	default:
		zap.S().Infof("Verifying command %v is in the channel's list.", trigger)
		available := false
		for _, comm := range ch.commands {
			if trigger == comm {
				available = true
				break
			}
		}
		if !available {
			zap.S().Infof("Couldn't find the %v command.", trigger)
			return ""
		}
		zap.S().Infof("Command is in the list, querying DB.")
		res, requiredPermission := CommandDBSelect(trigger, ch.database)
		if res == "" {
			zap.S().Infof("Couldn't find the %v command in the DB. This only happens if it was removed in the last 5 minutes.", trigger)
			result = "Command recently deleted."
		} else if !AuthorizeCommand(userPermissionLevel, userName, requiredPermission) {
			result = "Sorry, you're not authorized to use this command {user}."
		} else {
			result = res
		}
	}

	result = FormatResponse(result, message)

	return result
}
