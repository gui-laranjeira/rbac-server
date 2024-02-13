package main

import (
	"context"
	"log"

	"github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gui-laranjeira/rbac-server/internal/controllers"
	"github.com/gui-laranjeira/rbac-server/internal/middleware"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
)

var ctx context.Context
var err error
var client *mongo.Client
var _middleware *middleware.Middleware
var MongoUri string = "mongodb://mongo:27017/auth-server?authSource=admin"
var userController *controllers.UserController

func init() {
	// Mongo
	ctx = context.Background()
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(MongoUri))
	if err != nil {
		log.Fatalf("Error in connecting to database: %v", err)
		return
	}
	if err = client.Ping(context.TODO(), readpref.Primary()); err != nil {
		log.Fatalf("Error in connecting to database: %v", err)
		return
	}
	log.Println("Connected to MongoDB database")
	collection := client.Database("auth-server").Collection("users")

	// Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "root",
		DB:       0,
	})
	status := redisClient.Ping(ctx)
	if status.Err() != nil {
		log.Fatalf("Error while connecting to Redis: %v", status.Err())
		return
	}

	log.Printf("Connected to Redis: %v\n", status)

	_middleware = middleware.NewMiddleware(ctx, redisClient)
	userController = controllers.NewUserController(collection, ctx, redisClient)
}

func main() {
	app := fiber.New()
	logger, _ := zap.NewProduction()

	app.Use(fiberzap.New(fiberzap.Config{
		Logger: logger,
	}))

	app.Post("/signup", userController.CreateUser)
	app.Post("/addPermission", userController.AddPermission)
	app.Post("/login", userController.Login)
	app.Post("/adminTestRoute", _middleware.AdminMiddlewareHandler, userController.TestRoute)

	err := app.Listen(":8080")
	if err != nil {
		log.Fatal("Error in running server")
		return
	}
	log.Println("Server is running on port 8080")
}
