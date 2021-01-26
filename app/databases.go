// Package gotwitchbot contains a complete Twitch.tv bot, including IRC connection.
package main

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"

	"github.com/gempir/go-twitch-irc/v2"
)

const dbType = "postgres"

/* Secrets */

func getAWSSecret(secretName, region string) string {
	zap.S().Infof("Getting AWS Secret: %v", secretName)
	env := os.Getenv("ENV")
	serviceName := os.Getenv("SVCNAME")

	sess, err := session.NewSession(aws.NewConfig().WithRegion(region))
	if err != nil {
		handleAWSError(err)
	}
	svc := secretsmanager.New(sess)

	secretName = env + "/" + serviceName + "/" + secretName

	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := svc.GetSecretValue(input)
	if err != nil {
		handleAWSError(err)
	}
	return *result.SecretString
}

/* Commands */

func DoCommand(message twitch.PrivateMessage, ch broadcaster, re *regexp.Regexp) string {
	zap.S().Debugf("Executing a command")

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
		zap.S().Debug("User is the broadcaster")
		userLevel = "b"
	} else if userBadges["Moderater"] == 1 {
		zap.S().Debug("User is a moderator")
		userLevel = "m"
	} else {
		zap.S().Debug("User is a viewer")
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
	} else if trigger == "connectionTest" {
		permission := "m"
		if !AuthorizeCommand(userLevel, permission) {
			result = ""
		} else {
			result = "The bot has succesfully latched on to this channel."
		}
	} else if trigger == "joinchannel" {
		permission := "b"
		if !AuthorizeCommand(userLevel, permission) {
			result = ""
		} else {
			//BotDBBroadcasterAdd(broadcaster, botDB)
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
	zap.S().Debugf("Authorizing a command")
	if permissionLevel == "b" && userLevel != "b" {
		return false
	} else if permissionLevel == "m" || permissionLevel == "b" {
		return false
	} else {
		return true
	}
}

/* DB Functions */

func handleSQLError(err error) {
	zap.S().Errorf("SQL error: %v", err)
}

func DBConnect(dbEndpoint, dbUser, dbPassword, dbName, dbType string) (*sql.DB, error) {
	zap.S().Infof("Creating a DB Connection")
	dsn := fmt.Sprintf("postgres://%v/%v?sslmode=disable",
		dbEndpoint,
		dbName)

	u, err := url.Parse(dsn)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		panic(1)
	}

	zap.S().Infof("DB Url (no user): %v", u.String())

	u.User = url.UserPassword(dbUser, dbPassword)
	db, err := sql.Open(dbType, u.String())

	return db, err
}

/* Bot DB */

func BotDBPrepare() *sql.DB {
	zap.S().Infof("Preparing the bot DB")
	awsRegion := os.Getenv("AWS_REGION")
	endpoint := getAWSSecret("db-endpoint", awsRegion)
	dbUser := getAWSSecret("db-user", awsRegion)
	dbPassword := getAWSSecret("db-password", awsRegion)
	dbName := getAWSSecret("db-name", awsRegion)

	db, err := DBConnect(endpoint, dbUser, dbPassword, dbName, dbType)
	if err != nil {
		handleSQLError(err)
	}

	return db
}

func BotDBMainTablesPrepare(db *sql.DB) {
	zap.S().Info("Preparing the bot DB broadcasters table")
	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS broadcasters (channelname TEXT PRIMARY KEY, dbcreated BOOL);")
	if err != nil {
		handleSQLError(err)
	}
	statement.Exec()
}

func BotDBBroadcasterList(db *sql.DB) string {
	zap.S().Info("Listing broadcasters")
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
	zap.S().Info("Adding a new broadcaster")
	insertStatement := "INSERT INTO broadcasters (channelname, dbcreated) VALUES ('" + broadcaster + "', false) ON CONFLICT (channelname) DO NOTHING;"

	statement, err := db.Prepare(insertStatement)
	if err != nil {
		handleSQLError(err)
	}
	defer statement.Close()
	statement.Exec()

	zap.S().Infof("Checking if broadcaster %v is new / has a DB already", broadcaster)
	rows, err := db.Query("SELECT dbcreated FROM broadcasters WHERE channelname='" + broadcaster + "';")
	if err != nil {
		handleSQLError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var dbcreated bool
		if err := rows.Scan(&dbcreated); err != nil {
			handleSQLError(err)
		}
		if dbcreated == false {
			zap.S().Infof("dbcreated for %v is false", broadcaster)
			ChannelDBPrepare(db, broadcaster)
			stmnt, err := db.Prepare("UPDATE broadcasters SET dbcreated = true WHERE channelname = '" + broadcaster + "';")
			if err != nil {
				handleSQLError(err)
			}
			stmnt.Exec()
			zap.S().Infof("%v dbcreated set to true", broadcaster)
		}
	}

}

func BotDBBroadcasterRemove(broadcaster string, db *sql.DB) {
	zap.S().Info("Removing a broadcaster")
	deleteStatement := "DELETE FROM broadcasters WHERE channelname = '" + broadcaster + "';"

	statement, err := db.Prepare(deleteStatement)
	if err != nil {
		handleSQLError(err)
	}
	defer statement.Close()
	statement.Exec()
}

/* Channel DB */

func ChannelDBPrepare(botDB *sql.DB, channelName string) {
	zap.S().Infof("Preparing the %v channel DB", channelName)
	awsRegion := os.Getenv("AWS_REGION")
	dbUser := getAWSSecret("db-user", awsRegion)
	dbEndpoint := getAWSSecret("db-endpoint", awsRegion)
	dbPassword := getAWSSecret("db-password", awsRegion)
	command := fmt.Sprintf("CREATE DATABASE %s;", channelName)
	statement, err := botDB.Prepare(command)
	if err != nil {
		handleSQLError(err)
	}
	statement.Exec()

	zap.S().Info("Creating new DB conenction")
	database, err := DBConnect(dbEndpoint, dbUser, dbPassword, channelName, dbType)
	if err != nil {
		handleSQLError(err)
	}
	defer database.Close()

	CommandTablePrepare(database)
	UserDBPrepare(database)
}

func ChannelDBConnect(channelName string) *sql.DB {
	awsRegion := os.Getenv("AWS_REGION")
	dbUser := getAWSSecret("db-user", awsRegion)
	dbEndpoint := getAWSSecret("db-endpoint", awsRegion)
	dbPassword := getAWSSecret("db-password", awsRegion)

	database, err := DBConnect(dbEndpoint, dbUser, dbPassword, channelName, dbType)
	if err != nil {
		handleSQLError(err)
	}

	return database
}

/* Commands Table Interactions */

func CommandTablePrepare(db *sql.DB) {
	zap.S().Infof("Preparing the command table on a DB")
	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS commands (id INTEGER PRIMARY KEY, trigger TEXT UNIQUE, payload TEXT, permission TEXT, cooldown INTEGER, uses INTEGER);")
	if err != nil {
		handleSQLError(err)
	}
	defer statement.Close()
	statement.Exec()
}

func CommandDBSelect(trigger string, db *sql.DB) (string, string) {
	zap.S().Debugf("Triggering command: %v", trigger)
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
	zap.S().Info("Adding a command")
	statement, err := db.Prepare("INSERT INTO commands (trigger, payload, permission, cooldown) VALUES (?, ?, ?, ?);")
	if err != nil {
		panic(err)
	}
	defer statement.Close()
	statement.Exec(trigger, payload, permission, cooldown)

	return "Command " + trigger + " added succesfully."
}

func CommandDBRemove(trigger string, db *sql.DB) string {
	zap.S().Info("Removing a command")
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
	zap.S().Info("Preparing the User DB for a channel")
	// User table fields: Name, aliases, streams visited, last seen, watchtime, status, streamer BOOL, streamlink/shoutout
	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS channelusers (id INTEGER PRIMARY KEY, name TEXT, aliases BLOB, lastseen TEXT, streamsvisited INTEGER, watchtime INTEGER, streamer BOOL, streamlink TEXT)")
	if err != nil {
		panic(err)
	}
	defer statement.Close()
	statement.Exec()
}

func UserDBSelect(db *sql.DB) {
	zap.S().Info("Selecting from the user DB")
	fmt.Printf("%v\n", db)
}

func UserDBInsert(db *sql.DB) {
	zap.S().Info("Inserting into the User DB")
	fmt.Printf("%v\n", db)
}
