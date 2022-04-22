package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Account DB model
type Account struct {
	bun.BaseModel `bun:"table:accounts"`
	ID uuid.UUID `bun:",pk,type:uuid,default:gen_random_uuid()"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`

	// Relations
	Users []*User `bun:"rel:has-many,join:id=account_id"`
	Keys []*Key `bun:"rel:has-many,join:id=account_id"`
}

// Key DB model
type Key struct {
	bun.BaseModel `bun:"table:keys"`
	ID uuid.UUID `bun:",pk,type:uuid,default:gen_random_uuid()"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`

	// Relations
	AccountId uuid.UUID `bun:",type:uuid"`
	Account *Account `bun:"rel:belongs-to,join:account_id=id"`
}

// ====================
//        Setup
// ====================

func initAccountTables(db *bun.DB) {
	ctx := context.Background()
	db.NewCreateTable().IfNotExists().Model((*Account)(nil)).Exec(ctx)
	db.NewCreateTable().IfNotExists().Model((*Key)(nil)).Exec(ctx)
}

func (a *Account) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
		case *bun.UpdateQuery:
			a.UpdatedAt = time.Now()
	}
	return nil
}

func (a *Key) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
		case *bun.UpdateQuery:
			a.UpdatedAt = time.Now()
	}
	return nil
}

// ====================
//    Route Handlers
// ====================

// ====================
//     Middleware
// ====================

func requireAccount(c *fiber.Ctx, db *bun.DB) error {
	accountKey, err := getAccountKeyFromHeaders(c)
	if err != nil {
		fmt.Println(err)
		return errors.New("no account key provided")
	}

	key := new(Key)
	ctx := context.Background()
	err = db.NewSelect().Model(key).Where("id = ?", accountKey).Scan(ctx)

	if err != nil {
		fmt.Println(err)
		return errors.New("invalid account key")
	}

	return c.Next()
}

// ====================
//      Utilities
// ====================

func getAccountKeyFromHeaders(c *fiber.Ctx) (uuid.UUID, error) {
	headers := c.GetReqHeaders()
	return uuid.Parse(headers["Account-Key"])
}
