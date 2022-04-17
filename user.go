package main

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// User DB model
type User struct {
	bun.BaseModel `bun:"table:users"`
	ID uuid.UUID `bun:",pk,type:uuid,default:gen_random_uuid()"`
	Token string `bun:"-"`
	Username string // has idx
	Password string
	Role string
	Metadata map[string]interface{} `bun:"type:jsonb"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`

	// Relationships
	AccountId uuid.UUID `bun:",type:uuid"` // has idx
	Account *Account `bun:"rel:belongs-to,join:account_id=id"`
	Tokens []*Token `bun:"rel:has-many,join:id=user_id"`
}

// Client-facing User model
type PublicUser struct {
	ID uuid.UUID
	Token string
	Username string
	Role string
	Metadata map[string]interface{}
	CreatedAt time.Time
	UpdatedAt time.Time
}

func initUserTable(db *bun.DB) {
	ctx := context.Background()
	db.NewCreateTable().IfNotExists().Model((*User)(nil)).Exec(ctx)
}

var _ bun.BeforeAppendModelHook = (*User)(nil)
func (u *User) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
		case *bun.UpdateQuery:
			u.UpdatedAt = time.Now()
	}
	return nil
}

var _ bun.AfterCreateTableHook = (*User)(nil)
func (*User) AfterCreateTable(ctx context.Context, query *bun.CreateTableQuery) error {
	_, err := query.DB().NewCreateIndex().
		Model((*User)(nil)).
		Index("username_idx").
		IfNotExists().
		Column("username").
		Exec(ctx)

	if err != nil {
		return err
	}

	_, err = query.DB().NewCreateIndex().
		Model((*User)(nil)).
		Index("account_id_idx").
		IfNotExists().
		Column("account_id").
		Exec(ctx)

	return err
}

func initUserRoutes(app *fiber.App, db *bun.DB) {
	routes := app.Group("/api/v1/users", func(c *fiber.Ctx) error {
		return requireAdmin(c, db)
	})

	routes.Get("/", func(c *fiber.Ctx) error {
		return getHandler(c, db)
	})
	routes.Post("/", func(c *fiber.Ctx) error {
		return postHandler(c, db)
	})
	routes.Put("/:id", func(c *fiber.Ctx) error {
		return putHandler(c, db)
	})
	routes.Delete("/:id", func(c *fiber.Ctx) error {
		return deleteHandler(c, db)
	})
}

func getHandler(c *fiber.Ctx, db *bun.DB) error {
	ctx := context.Background()
	users := []User{}
	err := db.NewSelect().Model(&users).Scan(ctx)
	if err != nil {
		return err
	}

	publicUsers := []PublicUser{}
	for _, user := range users {
		publicUsers = append(publicUsers, *user.ToPublicUser())
	}

	return c.JSON(publicUsers)
}

func postHandler(c *fiber.Ctx, db *bun.DB) error {
	user := new(User)
	
	if err := c.BodyParser(user); err != nil {
		return err
	}

	if _, err := user.New(db); err != nil {
		return err
	}

	return c.JSON(user.ToPublicUser())
}

func (user *User) New(db *bun.DB) (sql.Result, error) {
	ctx := context.Background()

	if user.Username == "" || user.Password == "" {
		return nil, errors.New("no username or password")
	}

	found := new(User)
	db.NewSelect().Model(found).Where("username = ?", user.Username).Where("account_id = ?", user.AccountId).Scan(ctx)
	if found.Username == user.Username {
		return nil, errors.New("username in use")
	}

	user.ID = uuid.New()
	user.Password, _ = hashPassword(user.Password)

	return db.NewInsert().Model(user).Exec(ctx)
}

func (user *User) ToPublicUser() *PublicUser {
	publicUser := new(PublicUser)

	publicUser.ID = user.ID
	publicUser.Username = user.Username
	publicUser.Role = user.Role
	publicUser.Token = user.Token
	publicUser.Metadata = user.Metadata
	publicUser.CreatedAt = user.CreatedAt
	publicUser.UpdatedAt = user.UpdatedAt

	return publicUser
}

func putHandler(c *fiber.Ctx, db *bun.DB) error {
	ctx := context.Background()
	user := new(User)
	
	if err := c.BodyParser(user); err != nil {
		return err
	}

	if user.Password != "" {
		user.Password, _ = hashPassword(user.Password)
	}

	id := c.Params("id")
	_, err := db.NewUpdate().Model(user).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return err
	}

	return c.JSON(user.ToPublicUser())
}

func deleteHandler(c *fiber.Ctx, db *bun.DB) error {
	ctx := context.Background()

	id := c.Params("id")
	_, err := db.NewDelete().Model(new(User)).Where("id = ?", id).Exec(ctx)

	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"success": true})
}
