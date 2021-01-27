// Package gotwitchbot contains a complete Twitch.tv bot, including IRC connection.
package main

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

const dbType = "postgres"

// BOTDB is a global variable to hold the bot db connection since it's used all over
var BOTDB *sql.DB

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

func BotDBPrepare() {
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

	BOTDB = db
}

func BotDBMainTablesPrepare() {
	zap.S().Info("Preparing the bot DB broadcasters table")
	statement, err := BOTDB.Prepare("CREATE TABLE IF NOT EXISTS broadcasters (id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY, channelname TEXT UNIQUE, dbcreated BOOL, authorized BOOL);")
	if err != nil {
		handleSQLError(err)
	}
	statement.Exec()
}

func BotDBBroadcasterList() string {
	zap.S().Info("Listing broadcasters")
	rows, err := BOTDB.Query("SELECT channelname FROM broadcasters WHERE dbcreated=true")
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

func BotDBBroadcasterAdd(broadcaster string) {
	zap.S().Info("Adding a new broadcaster")
	insertStatement := "INSERT INTO broadcasters (channelname, dbcreated, authorized) VALUES ('" + broadcaster + "', false, false) ON CONFLICT (channelname) DO NOTHING;"

	statement, err := BOTDB.Prepare(insertStatement)
	if err != nil {
		handleSQLError(err)
	}
	defer statement.Close()
	statement.Exec()

	zap.S().Infof("Checking if %v is new / has a DB already", broadcaster)
	rows, err := BOTDB.Query("SELECT dbcreated, authorized FROM broadcasters WHERE channelname='" + broadcaster + "';")
	if err != nil {
		handleSQLError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			dbcreated  bool
			authorized bool
		)
		if err := rows.Scan(&dbcreated, &authorized); err != nil {
			handleSQLError(err)
		}
		if dbcreated == false && authorized == true {
			zap.S().Infof("dbcreated for %v is false and user is authorized", broadcaster)
			ChannelDBPrepare(broadcaster)
			stmnt, err := BOTDB.Prepare("UPDATE broadcasters SET dbcreated = true WHERE channelname = '" + broadcaster + "';")
			if err != nil {
				handleSQLError(err)
			}
			stmnt.Exec()
			zap.S().Infof("%v dbcreated set to true", broadcaster)
		}
	}
}

func BroadcasterAuthorize(broadcaster string) {
	zap.S().Infof("Authorizing %v", broadcaster)
	authorizeStatement, err := BOTDB.Prepare("UPDATE broadcasters SET authorized = true WHERE channelname = '" + broadcaster + "';")
	if err != nil {
		handleSQLError(err)
	}
	authorizeStatement.Exec()
	zap.S().Infof("%v authorized", broadcaster)
}

func BotDBBroadcasterRemove(broadcaster string) {
	zap.S().Info("Removing a broadcaster")
	deleteStatement := "DELETE FROM broadcasters WHERE channelname = '" + broadcaster + "';"

	statement, err := BOTDB.Prepare(deleteStatement)
	if err != nil {
		handleSQLError(err)
	}
	defer statement.Close()
	statement.Exec()
}

/* Channel DB */

func ChannelDBPrepare(channelName string) {
	zap.S().Infof("Preparing the %v channel DB", channelName)
	awsRegion := os.Getenv("AWS_REGION")
	dbUser := getAWSSecret("db-user", awsRegion)
	dbEndpoint := getAWSSecret("db-endpoint", awsRegion)
	dbPassword := getAWSSecret("db-password", awsRegion)
	command := fmt.Sprintf("CREATE DATABASE %s;", channelName)
	statement, err := BOTDB.Prepare(command)
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
	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS commands (id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY, trigger TEXT UNIQUE, payload TEXT, permission TEXT, cooldown INTEGER, uses INTEGER);")
	if err != nil {
		handleSQLError(err)
	}
	defer statement.Close()
	statement.Exec()
}

func CommandDBSelect(trigger string, db *sql.DB) (string, string) {
	zap.S().Debugf("Querying database for command command: %v", trigger)
	selectStatement := "SELECT payload, permission FROM commands WHERE trigger = '" + trigger + "';"

	rows, err := db.Query(selectStatement)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var payloadResult string
	var permissionResult string
	for rows.Next() {
		var (
			payload    string
			permission string
		)
		rows.Scan(&payload, &permission)
		zap.S().Debugf("Query result: payload: %v, permission: %v", payload, permission)
		payloadResult = payload
		permissionResult = permission
	}

	return payloadResult, permissionResult
}

func CommandDBInsert(trigger string, payload string, permission string, cooldown int, db *sql.DB) string {
	zap.S().Info("Adding a command")
	commandString := fmt.Sprintf("INSERT INTO commands (trigger, payload, permission, cooldown) VALUES ('%v', '%v', '%v', %v);", trigger, payload, permission, cooldown)
	statement, err := db.Prepare(commandString)
	if err != nil {
		handleSQLError(err)
		return "I couldn't add that command due to a SQL error."
	}
	defer statement.Close()
	statement.Exec()

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
	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS channelusers (id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY, name TEXT, aliases BLOB, lastseen TEXT, streamsvisited INTEGER, watchtime INTEGER, streamer BOOL, streamlink TEXT)")
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
