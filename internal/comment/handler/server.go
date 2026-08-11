package handler

import (
	"context"
	stderrors "errors"

	"go.uber.org/zap"

	commentv1 "github.com/BwCloudWeGo/bw-cli/api/gen/comment/v1"
	"github.com/BwCloudWeGo/bw-cli/internal/comment/dto"
	"github.com/BwCloudWeGo/bw-cli/internal/comment/entity"
	"github.com/BwCloudWeGo/bw-cli/internal/comment/service"
	apperrors "github.com/BwCloudWeGo/bw-cli/pkg/errors"
)

// Server 将 comment gRPC 请求适配到 service 用例。
type Server struct {
	commentv1.UnimplementedCommentServiceServer
	svc *service.Service
	log *zap.Logger
}

// NewServer 创建 comment gRPC 服务端适配器。
func NewServer(svc *service.Service, log *zap.Logger) *Server {
	if log == nil {
		log = zap.NewNop()
	}
	return &Server{svc: svc, log: log}
}

// CreateComment 处理创建 RPC。
func (s *Server) CreateComment(ctx context.Context, req *commentv1.CreateCommentRequest) (*commentv1.CommentResponse, error) {
	item, err := s.svc.Create(ctx, dto.CreateCommand{
		Name:        req.GetName(),
		Description: req.GetDescription(),
	})
	if err != nil {
		return nil, mapCommentError(err)
	}
	return toProto(item), nil
}

// GetComment 处理按 ID 查询。
func (s *Server) GetComment(ctx context.Context, req *commentv1.GetCommentRequest) (*commentv1.CommentResponse, error) {
	item, err := s.svc.Get(ctx, req.GetId())
	if err != nil {
		return nil, mapCommentError(err)
	}
	return toProto(item), nil
}

// ListComments 处理分页列表查询。
func (s *Server) ListComments(ctx context.Context, req *commentv1.ListCommentsRequest) (*commentv1.ListCommentsResponse, error) {
	list, err := s.svc.List(ctx, dto.ListCommand{
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	})
	if err != nil {
		return nil, mapCommentError(err)
	}
	resp := &commentv1.ListCommentsResponse{
		Items: make([]*commentv1.CommentResponse, 0, len(list.Items)),
		Total: list.Total,
	}
	for _, item := range list.Items {
		resp.Items = append(resp.Items, toProto(item))
	}
	return resp, nil
}

// UpdateComment 处理按 ID 更新。
func (s *Server) UpdateComment(ctx context.Context, req *commentv1.UpdateCommentRequest) (*commentv1.CommentResponse, error) {
	item, err := s.svc.Update(ctx, dto.UpdateCommand{
		ID:          req.GetId(),
		Name:        req.GetName(),
		Description: req.GetDescription(),
	})
	if err != nil {
		return nil, mapCommentError(err)
	}
	return toProto(item), nil
}

// DeleteComment 处理按 ID 删除。
func (s *Server) DeleteComment(ctx context.Context, req *commentv1.DeleteCommentRequest) (*commentv1.DeleteCommentResponse, error) {
	if err := s.svc.Delete(ctx, req.GetId()); err != nil {
		return nil, mapCommentError(err)
	}
	return &commentv1.DeleteCommentResponse{Success: true}, nil
}

func toProto(item *dto.CommentDTO) *commentv1.CommentResponse {
	return &commentv1.CommentResponse{
		Id:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func mapCommentError(err error) error {
	switch {
	case stderrors.Is(err, entity.ErrInvalidComment):
		return apperrors.InvalidArgument("invalid_comment", "invalid comment input")
	case stderrors.Is(err, entity.ErrCommentNotFound):
		return apperrors.NotFound("comment_not_found", "comment not found")
	default:
		return apperrors.Wrap(apperrors.KindInternal, "comment_service_error", "comment service error", err)
	}
}
