package main

import (
	"database/sql"
	"os"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

func initDb() (*bun.DB) {
	dsn := os.Getenv("DATABASE_URI")
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())
	
	initHooks(db)
	initTables(db)

	return db
}

func initTables(db *bun.DB) {
	initUserTable(db)
	initTokenTable(db)
}

func initHooks(db *bun.DB) {
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.FromEnv("BUNDEBUG"),
	))
}
