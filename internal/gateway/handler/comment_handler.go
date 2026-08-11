package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	commentv1 "github.com/BwCloudWeGo/bw-cli/api/gen/comment/v1"
	"github.com/BwCloudWeGo/bw-cli/internal/gateway/request"
	apperrors "github.com/BwCloudWeGo/bw-cli/pkg/errors"
	"github.com/BwCloudWeGo/bw-cli/pkg/httpx"
)

// CommentHandler 将 comment HTTP 接口适配到生成的 gRPC client。
type CommentHandler struct {
	client commentv1.CommentServiceClient
	log    *zap.Logger
}

// NewCommentHandler 将 comment gRPC client 注入 HTTP handler 方法。
func NewCommentHandler(client commentv1.CommentServiceClient, log *zap.Logger) *CommentHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &CommentHandler{
		client: client,
		log:    log,
	}
}

// Create 将 POST /api/v1/comments 代理到 CreateComment。
//func (h *CommentHandler) Create(c *gin.Context) {
//	var req request.CreateCommentRequest
//	if err := c.ShouldBindJSON(&req); err != nil {
//		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
//		return
//	}
//	resp, err := h.client.CreateComment(outgoingContext(c), &commentv1.CreateCommentRequest{
//		Name:        req.Name,
//		Description: req.Description,
//	})
//	if err != nil {
//		httpx.Error(c, apperrors.FromGRPC(err))
//		return
//	}
//	h.log.Info("gateway comment create proxied", zap.String("request_id", httpx.RequestID(c)), zap.String("aggregate_id", resp.GetId()))
//	httpx.Created(c, resp)
//}

// Get 将 GET /api/v1/comments/:id 代理到 GetComment。
func (h *CommentHandler) Get(c *gin.Context) {
	resp, err := h.client.GetComment(outgoingContext(c), &commentv1.GetCommentRequest{Id: c.Param("id")})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// List 将 GET /api/v1/comments 代理到 ListComments。
func (h *CommentHandler) List(c *gin.Context) {
	var req request.ListCommentRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.ListComments(outgoingContext(c), &commentv1.ListCommentsRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// Update 将 PUT /api/v1/comments/:id 代理到 UpdateComment。
func (h *CommentHandler) Update(c *gin.Context) {
	var req request.UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.UpdateComment(outgoingContext(c), &commentv1.UpdateCommentRequest{
		Id:          c.Param("id"),
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	h.log.Info("gateway comment update proxied", zap.String("request_id", httpx.RequestID(c)), zap.String("aggregate_id", resp.GetId()))
	httpx.OK(c, resp)
}

// Delete 将 DELETE /api/v1/comments/:id 代理到 DeleteComment。
func (h *CommentHandler) Delete(c *gin.Context) {
	resp, err := h.client.DeleteComment(outgoingContext(c), &commentv1.DeleteCommentRequest{Id: c.Param("id")})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	h.log.Info("gateway comment delete proxied", zap.String("request_id", httpx.RequestID(c)), zap.String("aggregate_id", c.Param("id")))
	httpx.OK(c, resp)
}
