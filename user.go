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
	ID uuid.UUID `bun:",pk,type:uuid"`
	Token string `bun:"-"`
	Email string `bun:",pk"`
	Password string
	Role string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Client-facing User model
type PublicUser struct {
	ID uuid.UUID
	Token string
	Email string
	Role string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func initUserTable(db *bun.DB) {
	ctx := context.Background()
	db.NewCreateTable().IfNotExists().Model((*User)(nil)).Exec(ctx)
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

	if user.Email == "" || user.Password == "" {
		return nil, errors.New("no email or password")
	}

	found := new(User)
	db.NewSelect().Model(found).Where("email = ?", user.Email).Scan(ctx)
	if found.Email == user.Email {
		return nil, errors.New("email in use")
	}

	user.ID = uuid.New()
	user.Password, _ = hashPassword(user.Password)

	return db.NewInsert().Model(user).Exec(ctx)
}

func (user *User) ToPublicUser() *PublicUser {
	publicUser := new(PublicUser)

	publicUser.ID = user.ID
	publicUser.Email = user.Email
	publicUser.Role = user.Role
	publicUser.Token = user.Token
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
