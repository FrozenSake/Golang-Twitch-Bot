// Package gotwitchbot contains a complete Twitch.tv bot, including IRC connection.
package gotwitchbot

const (
	// ircTwitch for Twitch's IRC constants.
	ircTwitchTLS = "irc://irc.chat.twitch.tv:6697"
	ircTwitch    = "irc://irc.chat.twitch.tv:6667"
	ircInvalid   = "421"

	// websocketTwitch for Twitch's websocket constants.
	webSocketTwitchTLS = "wss://irc-ws.chat.twitch.tv:443"
	webSocketTwitch    = "ws://irc-ws.chat.twitch.tv:80"

	// pingmMessage for Twitch's PING message as sourced from https://dev.twitch.tv/docs/irc/guide .
	pingMessage = "PING :tmi.twitch.tv"

	// CommandsCapability for Twitch's Commands: https://dev.twitch.tv/docs/irc/commands -- CAP REQ :twitch.tv/commands
	CommandsCapability = "twitch.tv/commands"

	// MembershipCapability for Twitch's Memberships: https://dev.twitch.tv/docs/irc/membership -- CAP REQ :twitch.tv/membership
	MembershipCapability = "twitch.tv/membership"

	// TagsCapability for Twitch's Tags: https://dev.twitch.tv/docs/irc/tags -- CAP REQ :twitch.tv/tags
	TagsCapability = "twitch.tv/membership"
)

func NewIRCClient(username, oauth string) {

}

func NewAnonymousIRCClient() {
	return NewIRCClient("justinfan1234321", "oauth:99999")
}

func Connect() {

}

func Join() {

}

func Part() {

}

func PrivMsg() {

}

func NewWebSocketClient(username, oauth string) {

}
