package db

import (
	"database/sql"
	_ "github.com/lib/pq"
	"gitlab.com/pbobby001/postit-scheduler/pkg/logs"
	"os"
)

var Connection *sql.DB

func Connect() {

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}

	Connection = db

	err = db.Ping()
	if err != nil {
		_ = logs.Logger.Critical("Unable to connect to database")
		panic(err)
	}

	logs.Logger.Info("Connected to Postgres DB successfully")
}

func Disconnect() {
	_ = logs.Logger.Warn("Attempting to disconnect from db....")
	err := Connection.Close()
	if err != nil {
		logs.Logger.Trace(err)
		return
	}
	logs.Logger.Info("Disconnected from db successfully...")
}
