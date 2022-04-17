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
}

func initAccountTable(db *bun.DB) {
	ctx := context.Background()
	db.NewCreateTable().IfNotExists().Model((*Account)(nil)).Exec(ctx)
}

var _ bun.BeforeAppendModelHook = (*Account)(nil)
func (a *Account) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
		case *bun.UpdateQuery:
			a.UpdatedAt = time.Now()
	}
	return nil
}

func requireAccount(c *fiber.Ctx, db *bun.DB) error {
	accountId, err := getAccountIdFromHeaders(c)
	if err != nil {
		fmt.Println(err)
		return errors.New("no account id provided")
	}

	account := new(Account)
	ctx := context.Background()
	err = db.NewSelect().Model(account).Where("id = ?", accountId).Scan(ctx)

	if err != nil {
		fmt.Println(err)
		return errors.New("invalid account id")
	}

	return c.Next()
}

func getAccountIdFromHeaders(c *fiber.Ctx) (uuid.UUID, error) {
	headers := c.GetReqHeaders()
	return uuid.Parse(headers["Account-Id"])
}
