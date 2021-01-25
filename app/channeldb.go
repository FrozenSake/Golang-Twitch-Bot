// Package gotwitchbot contains a complete Twitch.tv bot, including IRC connection.
package main

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsutils"

	"github.com/gempir/go-twitch-irc/v2"
)

const dbType = "postgres"

/* AWS Aurora DB */

func createInstance(config *aws.Config) *rds.RDS {
	botSession := session.Must(session.NewSession())

	// Create a RDS client with additional configuration
	svc := rds.New(botSession, config)

	return svc
}

func createConfig(region string) *aws.Config {
	config := aws.NewConfig().WithRegion(region)

	return config
}

/* Commands */

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
			result = CommandDBInsert(newTrigger, newOptions, level, ch.database, 0)
		}
	} else if trigger == "removecommand" {
		permission := "m"
		if !AuthorizeCommand(userLevel, permission) {
			result = "Sorry, you're not authorized to use this command {user}."
		} else {
			submatch = re.FindStringSubmatch(options)
			deleteTrigger := submatch[1]
			result = CommandDBRemove(deleteTrigger, ch.database)
		}
	} else {
		result, permission := CommandDBSelect(trigger, ch.database)
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

/* DB Functions */

func handleSQLError(err error) {
	fmt.Printf("SQL error: %v", err)
}

func DBConnect(dbEndpoint string, awsRegion string, dbUser string, dbName string, dbType string, awsCreds *credentials.Credentials) (*sql.DB, error) {
	builder := rdsutils.NewConnectionStringBuilder(dbEndpoint, awsRegion, dbUser, dbName, awsCreds)
	connectString, err := builder.WithTCPFormat().Build()
	if err != nil {
		handleAWSError(err)
		return nil, err
	}

	db, err := sql.Open(dbType, connectString)

	return db, err
}

/* Bot DB */

func BotDBPrepare() *sql.DB {
	// https://docs.aws.amazon.com/sdk-for-go/api/aws/credentials/#EnvProvider
	creds := credentials.NewEnvCredentials()
	endpoint := os.Getenv("RDS_ENDPOINT")
	awsRegion := os.Getenv("AWS_REGION")
	dbUser := os.Getenv("DB_USER")
	dbName := os.Getenv("DB_NAME")

	db, err := DBConnect(endpoint, awsRegion, dbUser, dbName, dbType, creds)
	if err != nil {
		handleSQLError(err)
	}

	return db
}

func BotDBMainTablesPrepare(db *sql.DB) {
	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS broadcasters (id INTERGER PRIMARY KEY, channelname TEXT UNIQUE)")
	if err != nil {
		handleSQLError(err)
	}
	statement.Exec()
}

func BotDBBroadcasterList(db *sql.DB) string {
	rows, err := db.Query("SELECT channelname FROM broadcasters")
	if err != nil {
		handleSQLError(err)
	}
	defer rows.Close()

	var result string
	for rows.Next() {
		var (
			channelName string
		)
		if err := rows.Scan(&channelName); err != nil {
			handleSQLError(err)
		}
		result += channelName + ";"
	}

	result = strings.TrimRight(result, ";")
	return result
}

func BotDBBroadcasterAdd(broadcaster string, db *sql.DB) {
	insertStatement := "INSERT INTO broadcasters (channelname) VALUES ('" + broadcaster + "') ON CONFLICT (channelname) DO NOTHING;"

	statement, err := db.Prepare(insertStatement)
	if err != nil {
		handleSQLError(err)
	}
	defer statement.Close()
	statement.Exec()
}

func BotDBBroadcasterRemove(broadcaster string, db *sql.DB) {
	deleteStatement := "DELETE FROM broadcasters WHERE channelname = '" + broadcaster + "';"

	statement, err := db.Prepare(deleteStatement)
	if err != nil {
		handleSQLError(err)
	}
	defer statement.Close()
	statement.Exec()
}

/* Channel DB */

func ChannelDBPrepare(botDB *sql.DB, channelName string) *sql.DB {
	creds := credentials.NewEnvCredentials()
	dbEndpoint := os.Getenv("RDS_ENDPOINT")
	awsRegion := os.Getenv("AWS_REGION")
	dbUser := os.Getenv("DB_USER")
	database, err := DBConnect(dbEndpoint, awsRegion, dbUser, channelName, dbType, creds)
	if err != nil {
		command := fmt.Sprintf("CREATE DATABASE %s", channelName)
		statement, err := botDB.Prepare(command)
		if err != nil {
			handleSQLError(err)
			return nil
		}
		statement.Exec()
		database, err = DBConnect(dbEndpoint, awsRegion, dbUser, channelName, dbType, creds)
	}
	CommandTablePrepare(database)
	UserDBPrepare(database)
	return database
}

/* Commands Table Interactions */

func CommandTablePrepare(db *sql.DB) {
	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS commands (id INTEGER PRIMARY KEY, trigger TEXT UNIQUE, payload TEXT, permission TEXT, cooldown INTEGER, uses INTEGER)")
	if err != nil {
		handleSQLError(err)
	}
	defer statement.Close()
	statement.Exec()
}

func CommandDBSelect(trigger string, db *sql.DB) (string, string) {
	selectStatement := "SELECT payload, permission FROM commands WHERE trigger = '" + trigger + "';"

	rows, err := db.Query(selectStatement)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var payload string
	var permission string
	for rows.Next() {
		rows.Scan(&payload, &permission)
	}

	return payload, permission
}

func CommandDBInsert(trigger string, payload string, permission string, db *sql.DB, cooldown int) string {
	statement, err := db.Prepare("INSERT INTO commands (trigger, payload, permission, cooldown) VALUES (?, ?, ?, ?)")
	if err != nil {
		panic(err)
	}
	defer statement.Close()
	statement.Exec(trigger, payload, permission, cooldown)

	return "Command " + trigger + " added succesfully."
}

func CommandDBRemove(trigger string, db *sql.DB) string {
	statement, err := db.Prepare("DELETE FROM commands WHERE trigger = '" + trigger + "';")
	if err != nil {
		panic(err)
	}
	defer statement.Close()
	statement.Exec()

	return "Command " + trigger + " removed succesfully."
}

/* User/Viewer Table Interactions */

func UserDBPrepare(db *sql.DB) {
	// User table fields: Name, aliases, streams visited, last seen, watchtime, status, streamer BOOL, streamlink/shoutout
	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS user (id INTEGER PRIMARY KEY, name TEXT, aliases BLOB, lastseen TEXT, streamsvisited INTEGER, watchtime INTEGER, streamer BOOL, streamlink TEXT)")
	if err != nil {
		panic(err)
	}
	defer statement.Close()
	statement.Exec()
}

func UserDBSelect(db *sql.DB) {
	fmt.Printf("%v\n", db)
}

func UserDBInsert(db *sql.DB) {
	fmt.Printf("%v\n", db)
}
