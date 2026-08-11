package handler

import (
	"fmt"
	"strconv"

	"github.com/BwCloudWeGo/bw-cli/internal/shop/entity"
	"github.com/BwCloudWeGo/bw-cli/pkg/alipayx"
	"github.com/BwCloudWeGo/bw-cli/pkg/config"
	"github.com/BwCloudWeGo/bw-cli/pkg/database"
	"github.com/BwCloudWeGo/bw-cli/pkg/middleware"
	"github.com/BwCloudWeGo/bw-cli/pkg/mysqlx"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	shopv1 "github.com/BwCloudWeGo/bw-cli/api/gen/shop/v1"
	"github.com/BwCloudWeGo/bw-cli/internal/gateway/request"
	apperrors "github.com/BwCloudWeGo/bw-cli/pkg/errors"
	"github.com/BwCloudWeGo/bw-cli/pkg/httpx"
)

// ShopHandler 将 shop HTTP 接口适配到生成的 gRPC client。
type ShopHandler struct {
	client shopv1.ShopServiceClient
	log    *zap.Logger
}

// NewShopHandler 将 shop gRPC client 注入 HTTP handler 方法。
func NewShopHandler(client shopv1.ShopServiceClient, log *zap.Logger) *ShopHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &ShopHandler{
		client: client,
		log:    log,
	}
}

// Create 将 POST /api/v1/shops 代理到 CreateShop。
func (h *ShopHandler) Create(c *gin.Context) {
	var req request.CreateShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.CreateShop(outgoingContext(c), &shopv1.CreateShopRequest{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	h.log.Info("gateway shop create proxied", zap.String("request_id", httpx.RequestID(c)), zap.String("aggregate_id", resp.GetId()))
	httpx.Created(c, resp)
}

// Get 将 GET /api/v1/shops/:id 代理到 GetShop。
func (h *ShopHandler) Get(c *gin.Context) {
	resp, err := h.client.GetShop(outgoingContext(c), &shopv1.GetShopRequest{Id: c.Param("id")})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// List 将 GET /api/v1/shops 代理到 ListShops。
func (h *ShopHandler) List(c *gin.Context) {
	var req request.ListShopRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.ListShops(outgoingContext(c), &shopv1.ListShopsRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// Update 将 PUT /api/v1/shops/:id 代理到 UpdateShop。
func (h *ShopHandler) Update(c *gin.Context) {
	var req request.UpdateShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.UpdateShop(outgoingContext(c), &shopv1.UpdateShopRequest{
		Id:          c.Param("id"),
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	h.log.Info("gateway shop update proxied", zap.String("request_id", httpx.RequestID(c)), zap.String("aggregate_id", resp.GetId()))
	httpx.OK(c, resp)
}

// Delete 将 DELETE /api/v1/shops/:id 代理到 DeleteShop。
func (h *ShopHandler) Delete(c *gin.Context) {
	resp, err := h.client.DeleteShop(outgoingContext(c), &shopv1.DeleteShopRequest{Id: c.Param("id")})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	h.log.Info("gateway shop delete proxied", zap.String("request_id", httpx.RequestID(c)), zap.String("aggregate_id", c.Param("id")))
	httpx.OK(c, resp)
}

//======================================================================

func (h *ShopHandler) SeedSms(c *gin.Context) {
	var req request.SeedSmsRequest
	if err := c.ShouldBind(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.SeedSms(outgoingContext(c), &shopv1.SeedSmsRequest{
		Phone: req.Phone,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

func (h *ShopHandler) Register(c *gin.Context) {
	var req request.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.Register(outgoingContext(c), &shopv1.RegisterRequest{
		Phone:      req.Phone,
		Code:       req.Code,
		Password:   req.Password,
		Nickname:   req.Nickname,
		ClientType: req.ClientType,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.Created(c, resp)
}

func (h *ShopHandler) Login(c *gin.Context) {
	var req request.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.Login(outgoingContext(c), &shopv1.LoginRequest{
		Phone:      req.Phone,
		Password:   req.Password,
		ClientType: req.ClientType,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}

	tokenV2, err := middleware.GenerateTokenV2(
		config.GlobalConfig.Middleware.JWT,
		middleware.JWTClaims{UserID: strconv.FormatInt(resp.UserId, 10)},
		req.ClientType,
	)
	if err != nil {
		return
	}

	httpx.OK(c, tokenV2)
}

func (h *ShopHandler) GetUser(c *gin.Context) {
	resp, err := h.client.GetUser(outgoingContext(c), &shopv1.GetUserRequest{
		Id: parseInt64Param(c, "id"),
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

func (h *ShopHandler) CreateArticle(c *gin.Context) {
	var req request.CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.CreateArticle(outgoingContext(c), &shopv1.CreateArticleRequest{
		UserId:  req.UserId,
		Title:   req.Title,
		Content: req.Content,
		Cover:   req.Cover,
		ShopId:  req.ShopId,
		Type:    req.Type,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.Created(c, resp)
}

func (h *ShopHandler) GetArticle(c *gin.Context) {
	resp, err := h.client.GetArticle(outgoingContext(c), &shopv1.GetArticleRequest{
		Id: parseInt64Param(c, "id"),
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

func (h *ShopHandler) ListArticles(c *gin.Context) {
	var req request.ListArticlesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.ListArticles(outgoingContext(c), &shopv1.ListArticlesRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

func (h *ShopHandler) UpdateArticle(c *gin.Context) {
	var req request.UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.UpdateArticle(outgoingContext(c), &shopv1.UpdateArticleRequest{
		Id:      parseInt64Param(c, "id"),
		Title:   req.Title,
		Content: req.Content,
		Cover:   req.Cover,
		Type:    req.Type,
		Status:  req.Status,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

func (h *ShopHandler) DeleteArticle(c *gin.Context) {
	resp, err := h.client.DeleteArticle(outgoingContext(c), &shopv1.DeleteArticleRequest{
		Id: parseInt64Param(c, "id"),
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

func (h *ShopHandler) SearchArticles(c *gin.Context) {
	var req request.SearchArticlesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.SearchArticles(outgoingContext(c), &shopv1.SearchArticlesRequest{
		Title:       req.Title,
		Content:     req.Content,
		LikeNumDesc: req.LikeNumDesc,
		Page:        req.Page,
		PageSize:    req.PageSize,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

func (h *ShopHandler) AddToCart(c *gin.Context) {
	var req request.AddToCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.AddToCart(outgoingContext(c), &shopv1.AddToCartRequest{
		UserId:    req.UserId,
		ProductId: req.ProductId,
		Num:       req.Num,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.Created(c, resp)
}

func (h *ShopHandler) CreateOrder(c *gin.Context) {
	var req request.CreateOrderRequest1
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.CreateOrder(outgoingContext(c), &shopv1.CreateOrderRequest{
		UserId:    req.UserId,
		ProductId: req.ProductId,
		Num:       req.Num,
		Address:   req.Address,
		PayMethod: req.PayMethod,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.Created(c, resp)
}

func (h *ShopHandler) CreateComment(c *gin.Context) {
	var req request.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.CreateComment(outgoingContext(c), &shopv1.CreateCommentRequest{
		UserId:    req.UserId,
		ArticleId: req.ArticleId,
		Content:   req.Content,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.Created(c, resp)
}

func (h *ShopHandler) GetArticleDetail(c *gin.Context) {
	var req request.GetArticleDetailRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.GetArticleDetail(outgoingContext(c), &shopv1.GetArticleDetailRequest{
		ArticleId: req.ArticleId,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

func (h *ShopHandler) AlipayNotify(c *gin.Context) {
	fmt.Println("===========================异步回调============================")
	err := c.Request.ParseForm()
	if err != nil {
		fmt.Println("解析参数失败")
		return
	}
	m := make(map[string]interface{})
	for k, v := range c.Request.Form {
		m[k] = v[0]
	}
	fmt.Println("回调后的数据是", m)
	fmt.Println("========================进行判断=================================")
	outNo := m["out_trade_no"]
	status := m["trade_status"]

	if outNo == "" {
		c.String(400, "订单号不存在")
		return
	}
	if status != "TRADE_SUCCESS" {
		c.String(400, "交易失败")
		return
	}
	fmt.Println("获取到订单号", outNo)
	fmt.Println("=====================进行支付宝验证签名=======================")
	alipay, err := alipayx.NewClient(config.GlobalConfig.Alipay)
	if err != nil {
		return
	}
	if err := alipay.VerifyReturn(c.Request.Context(), c.Request.Form); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("ALIPAY_SIGN_INVALID", err.Error()))
		return
	}
	fmt.Println("支付宝验签成功")
	fmt.Println("============================进行业务处理,修改订单状态为已支付==========================")
	db, err := mysqlx.Open(database.ToMySQLConfig(config.GlobalConfig.MySQL))
	if err != nil {
		return
	}
	var Order entity.Order
	Order.UpdateOrder(db, outNo)
	fmt.Println("业务处理成功")
	c.String(200, "success")
}
