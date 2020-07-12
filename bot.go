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

func DoCommand(message twitch.PrivateMessage, ch broadcaster, re *regexp.Regexp) string {

	///// REWORK TO INCLUDE command permission options structure.
	command := strings.ToLower(message.Message)
	submatch := re.FindStringSubmatch(command)
	trigger := submatch[1]
	level := submatch[2]
	options := submatch[3]
	var result string

	userBadges := message.User.Badges
	var userLevel string
	if userBadges["Broadcaster"] == 1 {
		userLevel = "b"
	} else if userBadges["Moderater"] == 1 {
		userLevel = "m"
	} else {
		userLevel = ""
	}

	if trigger == "addcommand" {
		permission := "m"
		if !AuthorizeCommand(userLevel, permission) {
			result = "Sorry, you're not authorized to use this command {user}."
		} else {
			submatch = re.FindStringSubmatch(options)
			newTrigger := submatch[1]
			newOptions := submatch[2]
			result = CommandDBInsert(newTrigger, newOptions, level, ch.name, 0)
		}
	} else if trigger == "removecommand" {
		permission := "m"
		if !AuthorizeCommand(userLevel, permission) {
			result = "Sorry, you're not authorized to use this command {user}."
		} else {
			submatch = re.FindStringSubmatch(options)
			deleteTrigger := submatch[1]
			result = CommandDBRemove(deleteTrigger, ch.name)
		}
	} else {
		result, permission := CommandDBSelect(trigger, ch.name)
		if result == "" {
			result = "No " + trigger + " command."
		}
		if !AuthorizeCommand(userLevel, permission) {
			result = "Sorry, you're not authorized to use this command {user}."
		}
	}

	result = FormatResponse(result, message)

	return result
}

func AuthorizeCommand(userLevel, permissionLevel string) bool {
	if permissionLevel == "b" && userLevel != "b" {
		return false
	} else if permissionLevel == "m" && (userLevel != "b" && userLevel != "m") {
		return false
	} else {
		return true
	}
}

func FormatResponse(payload string, message twitch.PrivateMessage) string {
	formatted := strings.ReplaceAll(payload, "{user}", message.User.DisplayName)
	formatted = strings.ReplaceAll(payload, "{target}", "{target}")

	return formatted
}

/* DB Functions */

func DBConnect(db string) *sql.DB {
	database, err := sql.Open("sqlite3", db)
	if err != nil {
		panic(err)
	}
	return database
}

/* Commands Table Interactions */

func CommandDBPrepare(channelName string) *sql.DB {
	dbname := "./" + channelName + ".db"

	database := DBConnect(dbname)

	statement, err := database.Prepare("CREATE TABLE IF NOT EXISTS commands (id INTEGER PRIMARY KEY, trigger TEXT UNIQUE, payload TEXT, permission TEXT, cooldown INTEGER, uses INTEGER)")
	if err != nil {
		panic(err)
	}
	statement.Exec()

	return database
}

func CommandDBSelect(trigger, channelName string) (string, string) {
	dbname := "./" + channelName + ".db"
	selectStatement := "SELECT payload, permission FROM commands WHERE trigger = '" + trigger + "';"

	database := DBConnect(dbname)

	rows, err := database.Query(selectStatement)
	if err != nil {
		panic(err)
	}

	var payload string
	var permission string
	for rows.Next() {
		rows.Scan(&payload, &permission)
	}

	return payload, permission
}

func CommandDBInsert(trigger, payload, permission, channelName string, cooldown int) string {
	dbname := "./" + channelName + ".db"

	database := DBConnect(dbname)

	statement, err := database.Prepare("INSERT INTO commands (trigger, payload, permission, cooldown) VALUES (?, ?, ?, ?)")
	if err != nil {
		panic(err)
	}
	statement.Exec(trigger, payload, permission, cooldown)

	return "Command " + trigger + " added succesfully."
}

func CommandDBRemove(trigger, channelName string) string {
	dbname := "./" + channelName + ".db"

	database := DBConnect(dbname)

	statement, err := database.Prepare("DELETE FROM commands WHERE trigger = '" + trigger + "';")
	if err != nil {
		panic(err)
	}
	statement.Exec()

	return "Command " + trigger + " removed succesfully."
}

/* User/Viewer Table Interactions */

func UserDBPrepare(channelName string) *sql.DB {
	// User table fields: Name, aliases, streams visited, last seen, watchtime, status, streamer BOOL, streamlink/shoutout
	dbname := "./" + channelName + ".db"

	database := DBConnect(dbname)

	statement, err := database.Prepare("CREATE TABLE IF NOT EXISTS user (id INTEGER PRIMARY KEY, name TEXT, aliases BLOB, lastseen TEXT, streamsvisited INTEGER, watchtime INTEGER, streamer BOOL, streamlink TEXT)")
	if err != nil {
		panic(err)
	}
	statement.Exec()

	return database
}

func UserDBSelect(channelName string) {
	dbname := "./" + channelName + ".db"

	database := DBConnect(dbname)

	fmt.Printf("%v\n", database)
}

func UserDBInsert(channelName string) {
	dbname := "./" + channelName + ".db"

	database := DBConnect(dbname)

	fmt.Printf("%v\n", database)
}

/* Run */

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
		DB := CommandDBPrepare(channelName)
		bc := broadcaster{name: channelName, database: DB}
		channels[channelName] = bc
	}

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		fmt.Printf("%v - %v: %v\n", message.Channel, message.User.DisplayName, message.Message)
		if re.MatchString(message.Message) {
			fmt.Println("##Possible Command detected!##")
			target := message.Channel
			command := DoCommand(message, channels[target], re)
			client.Say(target, command)
		}
	})

	err := client.Connect()
	if err != nil {
		panic(err)
	}
}
