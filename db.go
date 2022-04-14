package main

import (
	"context"
	"database/sql"
	"os"
	"time"

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

var _ bun.BeforeAppendModelHook = (*User)(nil)
func (u *User) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
		case *bun.InsertQuery:
			u.CreatedAt = time.Now()
			u.UpdatedAt = time.Now()
		case *bun.UpdateQuery:
			u.UpdatedAt = time.Now()
	}
	return nil
}

var _ bun.BeforeAppendModelHook = (*Token)(nil)
func (t *Token) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
		case *bun.InsertQuery:
			t.CreatedAt = time.Now()
			t.UpdatedAt = time.Now()
		case *bun.UpdateQuery:
			t.UpdatedAt = time.Now()
	}
	return nil
}

var _ bun.AfterCreateTableHook = (*Token)(nil)
func (*Token) AfterCreateTable(ctx context.Context, query *bun.CreateTableQuery) error {
	_, err := query.DB().NewCreateIndex().
		Model((*Token)(nil)).
		Index("value_idx").
		IfNotExists().
		Column("value").
		Exec(ctx)
	return err
}
