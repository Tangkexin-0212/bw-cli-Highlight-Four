package entity

import "context"

// Repository 定义 comment service 层需要的持久化行为。
type Repository interface {
	Save(ctx context.Context, item *Comment) error
	FindByID(ctx context.Context, id string) (*Comment, error)
	List(ctx context.Context, offset int, limit int) ([]*Comment, int64, error)
	Delete(ctx context.Context, id string) error
}
