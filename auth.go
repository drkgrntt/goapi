package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

// Token DB model
type Token struct {
	bun.BaseModel `bun:"table:tokens"`
	ID uuid.UUID `bun:",pk,type:uuid"`
	Value string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func initTokenTable(db *bun.DB) {
	ctx := context.Background()
	db.NewCreateTable().IfNotExists().Model((*Token)(nil)).Exec(ctx)
}

func initAuthRoutes(app *fiber.App, db *bun.DB) {
	routes := app.Group("/api/v1/auth")

	routes.Get("/", func(c *fiber.Ctx) error {
		return getCurrentUser(c, db)
	})

	routes.Post("/", func(c *fiber.Ctx) error {
		return register(c, db)
	})

	routes.Put("/", func(c *fiber.Ctx) error {
		return login(c, db)
	})

	routes.Delete("/", func(c *fiber.Ctx) error {
		return logout(c, db)
	})
}

func getCurrentUser(c *fiber.Ctx, db *bun.DB) error {
	headers := c.GetReqHeaders()
	tokenString := headers["X-Token"]

	if tokenString == "" {
		return c.JSON(nil)
	}

	user, err := getUserFromJwt(tokenString, db)
	if err != nil {
		fmt.Println(err)
		return c.JSON(nil)
	}

	return c.JSON(user.ToPublicUser())
}

func register(c *fiber.Ctx, db *bun.DB) error {
	user := new(User)
	
	if err := c.BodyParser(user); err != nil {
		return err
	}

	user.Role = ""
	_, err := user.New(db)

	if err != nil {
		return err
	}

	token := createJwt(user.ID.String(), db)
	user.Token = token
	
	return c.JSON(user.ToPublicUser())
}

func login(c * fiber.Ctx, db *bun.DB) error {
	ctx := context.Background()
	user := new(User)
	
	if err := c.BodyParser(user); err != nil {
		return err
	}

	found := new(User)
	db.NewSelect().Model(found).Where("email = ?", user.Email).Scan(ctx)

	match := checkPasswordHash(user.Password, found.Password)
	if !match || found.Password == "" {
		return errors.New("invalid email or password")
	}

	token := createJwt(found.ID.String(), db)
	found.Token = token

	return c.JSON(found.ToPublicUser())
}

func logout(c * fiber.Ctx, db *bun.DB) error {
	token := c.GetReqHeaders()["X-Token"]
	if token != "" {
		// Go through the token verification process
		// so that we can do nothing if invalid
		_, err := getUserFromJwt(token, db)
		if err == nil {
			// At this point, we're clear to delete the token
			ctx := context.Background()
			_, err := db.NewDelete().Model(new(Token)).Where("value = ?", unsignToken(token)).Exec(ctx)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			fmt.Println(err)
		}
	}

	// So as not to enumerate, always return success
	return c.JSON(fiber.Map{"success": true})
}

func createJwt(id string, db *bun.DB) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": id,
		"iss": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour*24*14).Unix(),
	})
	
	hmacSampleSecret := []byte(os.Getenv("JWT_SECRET"))

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(hmacSampleSecret)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	tokenRecord := new(Token)
	tokenRecord.Value = unsignToken(tokenString)
	tokenRecord.ID = uuid.New()

	db.NewInsert().Model(tokenRecord).Exec(ctx)

	return tokenString
}

func unsignToken(token string) string {
	pieces := strings.Split(token, ".")
	return strings.Join([]string{pieces[0], pieces[1]}, ".")
}

func getUserFromJwt(tokenString string, db *bun.DB) (*User, error) {
	ctx := context.Background()

	tokenObj := new(Token)
	err := db.NewSelect().Model(tokenObj).Where("value = ?", unsignToken(tokenString)).Scan(ctx)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	hmacSampleSecret := []byte(os.Getenv("JWT_SECRET"))
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return hmacSampleSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		
		user := new(User)
		err := db.NewSelect().Model(user).Where("id = ?", claims["id"]).Scan(ctx)
		if err != nil {
			return nil, err
		}

		return user, nil
	}

	return nil, errors.New("invalid token")
}

func requireAdmin(c * fiber.Ctx, db *bun.DB) error {
	headers := c.GetReqHeaders()
	tokenString := headers["X-Token"]
	if tokenString == "" {
		return errors.New("no token provided")
	}

	user, err := getUserFromJwt(tokenString, db)
	if err != nil {
		return err
	}

	if user.Role != "admin" {
		return errors.New("unauthorized")
	}

	return c.Next()
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
