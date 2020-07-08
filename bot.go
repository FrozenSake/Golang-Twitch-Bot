// Package gotwitchbot contains a complete Twitch.tv bot, including IRC connection.
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
	"github.com/joho/godotenv"
)

const (
	oauthForm = "oauth:"
)

var (
	username string
	oauth    string
	targets  []string
)

func goDotEnvVariable(key string) string {

	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	return os.Getenv(key)
}

func OauthCheck() {
	if oauth[:6] != oauthForm {
		oauth = oauthForm + oauth
	}
}

func main() {

	username = goDotEnvVariable("username")
	oauth = goDotEnvVariable("oauth")
	targetsLoad := goDotEnvVariable("channels")
	fmt.Println(targets)
	targets := strings.Split(targetsLoad, ",")
	fmt.Println(targets)

	OauthCheck()

	client := twitch.NewClient(username, oauth)

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		fmt.Printf("%v - %v: %v\n", message.Channel, message.User.DisplayName, message.Message)
		if message.Message[:1] == "!" {
			fmt.Println("##Command detected!##")
			target := message.Channel
			client.Say(target, "Hi, this is Hikthur testing.")
		}
	})

	for _, channel := range targets {
		client.Join(channel)
		fmt.Printf("##USERLIST FOR %v##\n", channel)
		userlist, _ := client.Userlist(channel)
		fmt.Printf("Users: %v\n", userlist)
	}

	err := client.Connect()
	if err != nil {
		panic(err)
	}
}
