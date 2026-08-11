package model

import "time"

// CommentModel 是 comments 表的 Gorm 持久化模型。
type CommentModel struct {
	ID          string `gorm:"primaryKey;size:64"`
	Name        string `gorm:"size:128;not null"`
	Description string `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (CommentModel) TableName() string {
	return "comments"
}

const commentMongoCollectionName = "comments"

// CommentDocument 是 comments 集合的 MongoDB 文档模型。
type CommentDocument struct {
	ID          string    `bson:"_id"`
	Name        string    `bson:"name"`
	Description string    `bson:"description"`
	CreatedAt   time.Time `bson:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at"`
}

func (CommentDocument) MongoCollectionName() string {
	return commentMongoCollectionName
}
