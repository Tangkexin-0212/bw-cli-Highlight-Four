package handler

import (
	"context"
	stderrors "errors"

	"go.uber.org/zap"

	shopv1 "github.com/BwCloudWeGo/bw-cli/api/gen/shop/v1"
	"github.com/BwCloudWeGo/bw-cli/internal/shop/dto"
	"github.com/BwCloudWeGo/bw-cli/internal/shop/entity"
	"github.com/BwCloudWeGo/bw-cli/internal/shop/service"
	apperrors "github.com/BwCloudWeGo/bw-cli/pkg/errors"
)

// Server 将 shop gRPC 请求适配到 service 用例。
type Server struct {
	shopv1.UnimplementedShopServiceServer
	svc *service.Service
	log *zap.Logger
}

// NewServer 创建 shop gRPC 服务端适配器。
func NewServer(svc *service.Service, log *zap.Logger) *Server {
	if log == nil {
		log = zap.NewNop()
	}
	return &Server{svc: svc, log: log}
}

// CreateShop 处理创建 RPC。
func (s *Server) CreateShop(ctx context.Context, req *shopv1.CreateShopRequest) (*shopv1.ShopResponse, error) {
	item, err := s.svc.Create(ctx, dto.CreateCommand{
		Name:        req.GetName(),
		Description: req.GetDescription(),
	})
	if err != nil {
		return nil, mapShopError(err)
	}
	return toProto(item), nil
}

// GetShop 处理按 ID 查询。
func (s *Server) GetShop(ctx context.Context, req *shopv1.GetShopRequest) (*shopv1.ShopResponse, error) {
	item, err := s.svc.Get(ctx, req.GetId())
	if err != nil {
		return nil, mapShopError(err)
	}
	return toProto(item), nil
}

// ListShops 处理分页列表查询。
func (s *Server) ListShops(ctx context.Context, req *shopv1.ListShopsRequest) (*shopv1.ListShopsResponse, error) {
	list, err := s.svc.List(ctx, dto.ListCommand{
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	})
	if err != nil {
		return nil, mapShopError(err)
	}
	resp := &shopv1.ListShopsResponse{
		Items: make([]*shopv1.ShopResponse, 0, len(list.Items)),
		Total: list.Total,
	}
	for _, item := range list.Items {
		resp.Items = append(resp.Items, toProto(item))
	}
	return resp, nil
}

// UpdateShop 处理按 ID 更新。
func (s *Server) UpdateShop(ctx context.Context, req *shopv1.UpdateShopRequest) (*shopv1.ShopResponse, error) {
	item, err := s.svc.Update(ctx, dto.UpdateCommand{
		ID:          req.GetId(),
		Name:        req.GetName(),
		Description: req.GetDescription(),
	})
	if err != nil {
		return nil, mapShopError(err)
	}
	return toProto(item), nil
}

// DeleteShop 处理按 ID 删除。
func (s *Server) DeleteShop(ctx context.Context, req *shopv1.DeleteShopRequest) (*shopv1.DeleteShopResponse, error) {
	if err := s.svc.Delete(ctx, req.GetId()); err != nil {
		return nil, mapShopError(err)
	}
	return &shopv1.DeleteShopResponse{Success: true}, nil
}

func toProto(item *dto.ShopDTO) *shopv1.ShopResponse {
	return &shopv1.ShopResponse{
		Id:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func mapShopError(err error) error {
	switch {
	case stderrors.Is(err, entity.ErrInvalidShop):
		return apperrors.InvalidArgument("invalid_shop", "invalid shop input")
	case stderrors.Is(err, entity.ErrShopNotFound):
		return apperrors.NotFound("shop_not_found", "shop not found")
	case stderrors.Is(err, entity.ErrProductNotFound):
		return apperrors.NotFound("product_not_found", "商品不存在")
	case stderrors.Is(err, entity.ErrUserNotAllowed):
		return apperrors.InvalidArgument("user_not_allowed", "用户状态异常，无法添加购物车")
	case stderrors.Is(err, entity.ErrProductNotOnSale):
		return apperrors.InvalidArgument("product_not_on_sale", "商品已下架，无法添加购物车")
	default:
		return apperrors.Wrap(apperrors.KindInternal, "shop_service_error", "shop service error", err)
	}
}

//============================================================================================

func (s *Server) SeedSms(ctx context.Context, req *shopv1.SeedSmsRequest) (*shopv1.SeedSmsResponse, error) {
	resp, err := s.svc.SeedSms(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Server) Register(ctx context.Context, req *shopv1.RegisterRequest) (*shopv1.RegisterResponse, error) {
	resp, err := s.svc.Register(ctx, req)
	if err != nil {
		return nil, mapShopError(err)
	}
	return resp, nil
}

func (s *Server) Login(ctx context.Context, req *shopv1.LoginRequest) (*shopv1.LoginResponse, error) {
	resp, err := s.svc.Login(ctx, req)
	if err != nil {
		return nil, mapShopError(err)
	}
	return resp, nil
}

func (s *Server) GetUser(ctx context.Context, req *shopv1.GetUserRequest) (*shopv1.UserResponse, error) {
	resp, err := s.svc.GetUser(ctx, req)
	if err != nil {
		return nil, mapShopError(err)
	}
	return resp, nil
}

func (s *Server) CreateArticle(ctx context.Context, req *shopv1.CreateArticleRequest) (*shopv1.ArticleResponse, error) {
	resp, err := s.svc.CreateArticle(ctx, req)
	if err != nil {
		return nil, mapShopError(err)
	}
	return resp, nil
}

func (s *Server) GetArticle(ctx context.Context, req *shopv1.GetArticleRequest) (*shopv1.ArticleResponse, error) {
	resp, err := s.svc.GetArticle(ctx, req)
	if err != nil {
		return nil, mapShopError(err)
	}
	return resp, nil
}

func (s *Server) ListArticles(ctx context.Context, req *shopv1.ListArticlesRequest) (*shopv1.ListArticlesResponse, error) {
	resp, err := s.svc.ListArticles(ctx, req)
	if err != nil {
		return nil, mapShopError(err)
	}
	return resp, nil
}

func (s *Server) UpdateArticle(ctx context.Context, req *shopv1.UpdateArticleRequest) (*shopv1.ArticleResponse, error) {
	resp, err := s.svc.UpdateArticle(ctx, req)
	if err != nil {
		return nil, mapShopError(err)
	}
	return resp, nil
}

func (s *Server) DeleteArticle(ctx context.Context, req *shopv1.DeleteArticleRequest) (*shopv1.DeleteArticleResponse, error) {
	resp, err := s.svc.DeleteArticle(ctx, req)
	if err != nil {
		return nil, mapShopError(err)
	}
	return resp, nil
}

func (s *Server) SearchArticles(ctx context.Context, req *shopv1.SearchArticlesRequest) (*shopv1.ListArticlesResponse, error) {
	resp, err := s.svc.SearchArticles(ctx, req)
	if err != nil {
		return nil, mapShopError(err)
	}
	return resp, nil
}

func (s *Server) AddToCart(ctx context.Context, req *shopv1.AddToCartRequest) (*shopv1.AddToCartResponse, error) {
	resp, err := s.svc.AddToCart(ctx, req)
	if err != nil {
		return nil, mapShopError(err)
	}
	return resp, nil
}

func (s *Server) CreateOrder(ctx context.Context, req *shopv1.CreateOrderRequest) (*shopv1.CreateOrderResponse, error) {
	resp, err := s.svc.CreateOrder(ctx, req)
	if err != nil {
		return nil, mapShopError(err)
	}
	return resp, nil
}

func (s *Server) CreateComment(ctx context.Context, req *shopv1.CreateCommentRequest) (*shopv1.CreateCommentResponse, error) {
	resp, err := s.svc.CreateComment(ctx, req)
	if err != nil {
		return nil, mapShopError(err)
	}
	return resp, nil
}

func (s *Server) GetArticleDetail(ctx context.Context, req *shopv1.GetArticleDetailRequest) (*shopv1.ArticleDetailResponse, error) {
	resp, err := s.svc.GetArticleDetail(ctx, req)
	if err != nil {
		return nil, mapShopError(err)
	}
	return resp, nil
}
