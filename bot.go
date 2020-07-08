// Package gotwitchbot contains a complete Twitch.tv bot, including IRC connection.
package gotwitchbot

var (
	username string = os.GetEnv("TWITCHID")
	oauth    string = os.GetEnv("OAUTH")
)

func main() {
	client := gotwitchbot.NewIRCClient(username, oauth)
	client.Connect()
}
