package request

// CreateShopRequest 是 POST /api/v1/shops 使用的 JSON 载荷。
type CreateShopRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateShopRequest 是 PUT /api/v1/shops/:id 使用的 JSON 载荷。
type UpdateShopRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// ListShopRequest 是 GET /api/v1/shops 使用的查询参数载荷。
type ListShopRequest struct {
	Page     int32 `form:"page"`
	PageSize int32 `form:"page_size"`
}

//=================================================================

type SeedSmsRequest struct {
	Phone string `form:"phone" json:"phone" xml:"phone"  binding:"required"`
}

// 用户
type RegisterRequest struct {
	Phone      string `json:"phone" binding:"required"`
	Code       string `json:"code" binding:"required"`
	Password   string `json:"password" binding:"required"`
	Nickname   string `json:"nickname"`
	ClientType string `json:"client_type"`
}

type LoginRequest struct {
	Phone      string `json:"phone" binding:"required"`
	Password   string `json:"password" binding:"required"`
	ClientType string `json:"client_type"`
}

// 文章
type CreateArticleRequest struct {
	UserId  int64  `json:"user_id" binding:"required"`
	Title   string `json:"title" binding:"required"`
	Content string `json:"content"`
	Cover   string `json:"cover"`
	ShopId  int64  `json:"shop_id"`
	Type    int64  `json:"type"`
}

type UpdateArticleRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Cover   string `json:"cover"`
	Type    int64  `json:"type"`
	Status  int64  `json:"status"`
}

type ListArticlesRequest struct {
	Page     int32 `form:"page"`
	PageSize int32 `form:"page_size"`
}

type SearchArticlesRequest struct {
	Title       string `form:"title"`
	Content     string `form:"content"`
	LikeNumDesc int64  `form:"like_num_desc"`
	Page        int64  `form:"page"`
	PageSize    int64  `form:"page_size"`
}

// 购物车
type AddToCartRequest struct {
	UserId    int64 `json:"user_id" binding:"required"`
	ProductId int64 `json:"product_id" binding:"required"`
	Num       int64 `json:"num" binding:"required"`
}

// 下单
type CreateOrderRequest1 struct {
	UserId    int64  `json:"user_id" binding:"required"`
	ProductId int64  `json:"product_id" binding:"required"`
	Num       int64  `json:"num" binding:"required"`
	Address   string `json:"address" binding:"required"`
	PayMethod int64  `json:"pay_method"`
}

// 评论
type CreateCommentRequest struct {
	UserId    int64 `json:"user_id" binding:"required"`
	ArticleId int64 `json:"article_id" binding:"required"`
	Content   string `json:"content" binding:"required"`
}

// 文章聚合详情
type GetArticleDetailRequest struct {
	ArticleId int64 `form:"article_id" binding:"required"`
}
