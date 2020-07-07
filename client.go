// Package gotwitchbot contains a complete Twitch.tv bot, including IRC connection.
package gotwitchbot

import (

)

const (
	ircTwitchTLS = "irc://irc.chat.twitch.tv:6697"
	ircTwitch    = "irc://irc.chat.twitch.tv:6667"
	ircInvalid   = "421"

	webSocketTwitchTLS = "wss://irc-ws.chat.twitch.tv:443"
	webSocketTwitch    = "ws://irc-ws.chat.twitch.tv:80"

	pingMessage = "PING :tmi.twitch.tv"

	// https://dev.twitch.tv/docs/irc/commands -- CAP REQ :twitch.tv/commands
	CommandsCapability = "twitch.tv/commands"

	// https://dev.twitch.tv/docs/irc/membership -- CAP REQ :twitch.tv/membership
	MembershipCapability = "twitch.tv/membership"

	// https://dev.twitch.tv/docs/irc/tags -- CAP REQ :twitch.tv/tags
	TagsCapability = "twitch.tv/membership"
)

func NewIRCClient(username, oauth string) {

}

func NewAnonymousIRCClient() {
	return NewIRCClient("justinfan1234321", "oauth:99999")
}

func Join {

}

func Part {

}

func PrivMsg {

}

func NewWebSocketClient (username, oauth string) {

}