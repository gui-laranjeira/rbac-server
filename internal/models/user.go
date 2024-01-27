package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Username    string             `bson:"username" json:"username"`
	Password    string             `bson:"password" json:"password"`
	Permissions []Permissions      `bson:"permissions" json:"permissions"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}

type Permissions struct {
	Entry      int  `bson:"entry" json:"entry"`
	AddFlag    bool `bson:"add_flag" json:"add_flag"`
	AdminFlag bool `bson:"admin_flag" json:"admin_flag"`
}

