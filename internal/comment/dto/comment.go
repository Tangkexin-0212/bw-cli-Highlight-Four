package dto

import (
	"time"

	"github.com/BwCloudWeGo/bw-cli/internal/comment/entity"
)

// CommentDTO 由用例返回，并由 handler 转换。
type CommentDTO struct {
	ID          string
	Name        string
	Description string
	CreatedAt   string
	UpdatedAt   string
}

// ListCommentDTO 包含分页列表出参。
type ListCommentDTO struct {
	Items []*CommentDTO
	Total int64
}

// FromComment 将 comment 聚合转换为 service 响应 DTO。
func FromComment(item *entity.Comment) *CommentDTO {
	return &CommentDTO{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		CreatedAt:   formatTime(item.CreatedAt),
		UpdatedAt:   formatTime(item.UpdatedAt),
	}
}

// formatTime 让零值时间保持为空，并用稳定 API 格式序列化真实时间。
func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}
