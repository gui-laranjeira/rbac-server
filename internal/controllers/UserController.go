package controllers

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gui-laranjeira/rbac-server/internal/models"
	"github.com/gui-laranjeira/rbac-server/internal/utils"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserController struct {
	collection  *mongo.Collection
	ctx         context.Context
	redisClient *redis.Client
}

type SingupInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AddPermissionInput struct {
	Username   string             `json:"username"`
	Permission models.Permissions `json:"permission"`
}

func NewUserController(collection *mongo.Collection, ctx context.Context, redisClient *redis.Client) *UserController {
	return &UserController{collection: collection, ctx: ctx, redisClient: redisClient}
}

func (u *UserController) CreateUser(c *fiber.Ctx) error {
	singupRequest := new(SingupInput)

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

func (u *UserController) AddPermission(c *fiber.Ctx) error {
	addPermissionInput := new(AddPermissionInput)
	if err := c.BodyParser(addPermissionInput); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Bad Request",
		})
	}

	user := new(models.User)

	err := u.collection.FindOne(u.ctx, bson.D{{"username", addPermissionInput.Username}}).Decode(&user)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "User not found",
		})
	}
	log.Println("User found: ", user)

	for _, v := range user.Permissions {
		if v.Entry == addPermissionInput.Permission.Entry {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"message": "Permission already exists",
			})
		}
	}

	u.collection.FindOneAndUpdate(u.ctx, bson.D{{"username", addPermissionInput.Username}}, bson.M{"$push": bson.M{"permissions": addPermissionInput.Permission}})
	return c.JSON(fiber.Map{"message": "Permission added successfully"})
}
