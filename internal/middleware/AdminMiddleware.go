package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber/v2"
	"github.com/gui-laranjeira/rbac-server/internal/models"
	"github.com/redis/go-redis/v9"
)

var SecretKey = []byte("secret")

type Middleware struct {
	ctx         context.Context
	redisClient *redis.Client
}

func NewMiddleware(ctx context.Context, redisClient *redis.Client) *Middleware {
	return &Middleware{
		ctx:         ctx,
		redisClient: redisClient,
	}
}

func (m *Middleware) AdminMiddlewareHandler(c *fiber.Ctx) error {
	authorization := c.Get("Authorization")
	entry := c.Get("Entry")
	entryInt, _ := strconv.Atoi(entry)
	token, err := jwt.Parse(authorization, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid signing method")
		}
		return SecretKey, nil
	})
	if err != nil {
		log.Println("Token parsing error: ", err.Error())
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	var hash string

	if token.Valid {
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			hash = claims["hash"].(string)
			expirationAsFloat := claims["exp"].(float64)

			if !ok {
				log.Println("Error parsing claims")
				return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
			}

			expiration := time.Unix(int64(expirationAsFloat), 0)

			if time.Now().After(expiration) {
				log.Println("Token expired")
				return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
			}
		}
	} else {
		log.Println("Token invalid")
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	value, _ := m.redisClient.Get(m.ctx, hash).Result()
	var retrievedPermissions []models.Permissions
	if err := json.Unmarshal([]byte(value), &retrievedPermissions); err != nil {
		log.Println("Error unmarshalling permissions: ", err)
		return err
	}
	log.Println("Retrieved permissions: ", retrievedPermissions)

	for _, permission := range retrievedPermissions {
		if permission.Entry == entryInt {
			if permission.AddFlag {
				return c.Next()
			} else {
				return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
			}

		} else {
			return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
		}
	}
	return nil
}
