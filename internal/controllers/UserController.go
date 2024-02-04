package controllers

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gui-laranjeira/rbac-server/internal/models"
	"github.com/gui-laranjeira/rbac-server/internal/utils"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserController struct {
	collection  *mongo.Collection
	ctx         context.Context
	redisClient *redis.Client
}

type Singup struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewUserController(collection *mongo.Collection, ctx context.Context, redisClient *redis.Client) *UserController {
	return &UserController{collection: collection, ctx: ctx, redisClient: redisClient}
}

func (u *UserController) CreateUser(c *fiber.Ctx) error {
	singupRequest := new(Singup)

	if err := c.BodyParser(singupRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Bad Request",
		})
	}

	hashedPass, err := utils.HashPassword(singupRequest.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Internal Server Error",
		})
	}

	user := new(models.User)
	user.ID = primitive.NewObjectID()
	user.Username = singupRequest.Username
	user.Password = hashedPass
	user.Permissions = make([]models.Permissions, 0)

	savedUser, err := u.collection.InsertOne(u.ctx, user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Internal Server Error",
		})
	}
	log.Println("User created: ", savedUser)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "User created successfully"})
}
