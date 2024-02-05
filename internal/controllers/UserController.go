package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"
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

var SecretKey = []byte("secret")

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
	tempUser := new(models.User)
	err = u.collection.FindOne(u.ctx, bson.D{{Key: "username", Value: singupRequest.Username}}).Decode(&tempUser)
	if err == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "User already exists",
		})
	}
	user := new(models.User)
	user.ID = primitive.NewObjectID()
	user.Username = singupRequest.Username
	user.Password = hashedPass
	user.Permissions = make([]models.Permissions, 0)
	user.CreatedAt = time.Now()

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

	err := u.collection.FindOne(u.ctx, bson.D{{Key: "username", Value: addPermissionInput.Username}}).Decode(&user)
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

	u.collection.FindOneAndUpdate(u.ctx, bson.D{{Key: "username", Value: addPermissionInput.Username}}, bson.M{"$push": bson.M{"permissions": addPermissionInput.Permission}})
	return c.JSON(fiber.Map{"message": "Permission added successfully"})
}

func (u *UserController) Login(c *fiber.Ctx) error {
	signupInput := new(SingupInput)

	if err := c.BodyParser(signupInput); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Bad Request",
		})
	}

	user := new(models.User)

	err := u.collection.FindOne(u.ctx, bson.D{{Key: "username", Value: signupInput.Username}}).Decode(&user)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "User not found",
		})
	}

	err = utils.VerifyPassword(signupInput.Password, user.Password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Invalid Password",
		})
	}

	data := []byte(fmt.Sprintf("%+v", user.Permissions))
	hasher := sha256.New()

	_, err = hasher.Write(data)
	if err != nil {
		log.Println("Failed to hash permissions")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Internal Server Error",
		})
	}
	hash := hasher.Sum(nil)
	hashString := hex.EncodeToString(hash)

	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["hash"] = hashString
	claims["exp"] = time.Now().Add(time.Hour * 1).Unix()

	permissionsJSON, err := json.Marshal(user.Permissions)
	if err != nil {
		log.Println("Failed to marshal permissions")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to marshal permissions",
		})
	}

	//Caching user permissions in redis
	result, err := u.redisClient.SetNX(u.ctx, hashString, permissionsJSON, time.Hour*1).Result()
	log.Println("Redis set result: ", result)
	log.Println("Error: ", err)

	tokenString, err := token.SignedString(SecretKey)
	if err != nil {
		log.Println("Failed to sign token")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Internal Server Error",
		})
	}
	log.Println("JWT Token: ", tokenString)
	return c.JSON(fiber.Map{"token": tokenString})
}

func (u *UserController) TestRoute(c *fiber.Ctx) error {
	return c.SendString("Admin Hello, World 👋!")
}
