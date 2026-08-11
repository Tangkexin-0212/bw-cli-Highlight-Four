package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/BwCloudWeGo/bw-cli/internal/comment/dto"
	"github.com/BwCloudWeGo/bw-cli/internal/comment/entity"
)

// Service 编排 comment 用例。
type Service struct {
	repo entity.Repository
	log  *zap.Logger
}

// NewService 创建 comment 用例服务。
func NewService(repo entity.Repository, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{repo: repo, log: log}
}

// Create 创建 comment 记录。
func (s *Service) Create(ctx context.Context, cmd dto.CreateCommand) (*dto.CommentDTO, error) {
	item, err := entity.NewComment(cmd.Name, cmd.Description)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, item); err != nil {
		return nil, err
	}
	s.log.Info("comment created", zap.String("aggregate_id", item.ID), zap.String("use_case", "CreateComment"))
	return dto.FromComment(item), nil
}

// Get 根据 ID 返回一条 comment 记录。
func (s *Service) Get(ctx context.Context, id string) (*dto.CommentDTO, error) {
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.FromComment(item), nil
}

// List 返回分页的 comment 记录。
func (s *Service) List(ctx context.Context, cmd dto.ListCommand) (*dto.ListCommentDTO, error) {
	offset, limit := normalizePagination(cmd.Page, cmd.PageSize)
	items, total, err := s.repo.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	output := &dto.ListCommentDTO{Items: make([]*dto.CommentDTO, 0, len(items)), Total: total}
	for _, item := range items {
		output.Items = append(output.Items, dto.FromComment(item))
	}
	return output, nil
}

// Update 根据 ID 修改一条 comment 记录。
func (s *Service) Update(ctx context.Context, cmd dto.UpdateCommand) (*dto.CommentDTO, error) {
	item, err := s.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if err := item.Update(cmd.Name, cmd.Description); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, item); err != nil {
		return nil, err
	}
	s.log.Info("comment updated", zap.String("aggregate_id", cmd.ID), zap.String("use_case", "UpdateComment"))
	return dto.FromComment(item), nil
}

// Delete 根据 ID 删除一条 comment 记录。
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.log.Info("comment deleted", zap.String("aggregate_id", id), zap.String("use_case", "DeleteComment"))
	return nil
}

func normalizePagination(page int32, pageSize int32) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return int((page - 1) * pageSize), int(pageSize)
}
