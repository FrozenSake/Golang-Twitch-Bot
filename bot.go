// Package gotwitchbot contains a complete Twitch.tv bot, including IRC connection.
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/gempir/go-twitch-irc/v2"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

const (
	oauthForm = "oauth:"
	// First group is command, second group is payload
	commandRegex = "^!(\\S+) ?(.*)"
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

func DoCommand(command string, ch broadcaster, re *regexp.Regexp) string {
	submatch := re.FindStringSubmatch(command)
	trigger := strings.ToLower(submatch[1])
	options := submatch[2]
	var result string

	if trigger == "addcommand" {
		result = DBCommandInsert(trigger, options, ch.name)
	} else {
		result = DBCommandSelect(trigger, ch.name)
	}

	return result
}

func DBPrepare(channelName string) *sql.DB {
	dbname := "./" + channelName + ".db"

	database, err := sql.Open("sqlite3", dbname)
	if err != nil {
		panic(err)
	}

	statement, err := database.Prepare("CREATE TABLE IF NOT EXISTS commands (id INTEGER PRIMARY KEY, trigger TEXT UNIQUE, payload TEXT)")
	if err != nil {
		panic(err)
	}
	statement.Exec()

	return database
}

func DBCommandSelect(trigger, channelName string) string {
	dbname := "./" + channelName + ".db"
	selectStatement := "SELECT payload FROM commands WHERE trigger = '" + trigger + "';"

	database, err := sql.Open("sqlite3", dbname)
	if err != nil {
		panic(err)
	}

	fmt.Println("Before DB Query with statement: ", selectStatement)
	rows, err := database.Query(selectStatement)
	if err != nil {
		panic(err)
	}
	fmt.Println("After DB Query")

	var payload string
	for rows.Next() {
		rows.Scan(&payload)
	}

	return payload
}

func DBCommandInsert(trigger, payload, channelName string) string {
	dbname := "./" + channelName + ".db"
	database, err := sql.Open("sqlite3", dbname)
	if err != nil {
		panic(err)
	}

	statement, err := database.Prepare("INSERT INTO commands (trigger, payload) VALUES (?, ?)")
	if err != nil {
		panic(err)
	}
	statement.Exec(trigger, payload)

	return "Command " + trigger + " added succesfully."
}

func main() {

	username = goDotEnvVariable("username")
	oauth = goDotEnvVariable("oauth")
	targetsLoad := goDotEnvVariable("channels")
	fmt.Println(targets)
	targets := strings.Split(targetsLoad, ",")
	fmt.Println(targets)

	OauthCheck()
	channels = make(map[string]broadcaster)

	// Define a regex object
	re := regexp.MustCompile(commandRegex)

	client := twitch.NewClient(username, oauth)

	for _, channelName := range targets {
		channelName = strings.ToLower(channelName)
		client.Join(channelName)
		fmt.Printf("##USERLIST FOR %v##\n", channelName)
		userlist, _ := client.Userlist(channelName)
		fmt.Printf("Users: %v\n", userlist)
		DB := DBPrepare(channelName)
		bc := broadcaster{name: channelName, database: DB}
		channels[channelName] = bc
	}

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		fmt.Printf("%v - %v: %v\n", message.Channel, message.User.DisplayName, message.Message)
		if re.MatchString(message.Message) {
			fmt.Println("##Possible Command detected!##")
			target := message.Channel
			command := DoCommand(message.Message, channels[target], re)
			fmt.Printf("Command is: %v", command)
			client.Say(target, command)
		}
	})

	err := client.Connect()
	if err != nil {
		panic(err)
	}
}
