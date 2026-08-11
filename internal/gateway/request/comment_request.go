package request

// CreateCommentRequest 是 POST /api/v1/comments 使用的 JSON 载荷。
//type CreateCommentRequest struct {
//	Name        string `json:"name" binding:"required"`
//	Description string `json:"description"`
//}

// UpdateCommentRequest 是 PUT /api/v1/comments/:id 使用的 JSON 载荷。
type UpdateCommentRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// ListCommentRequest 是 GET /api/v1/comments 使用的查询参数载荷。
type ListCommentRequest struct {
	Page     int32 `form:"page"`
	PageSize int32 `form:"page_size"`
}
