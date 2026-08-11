package entity

import "context"

// Repository 定义 shop service 层需要的持久化行为。
type Repository interface {
	Save(ctx context.Context, item *Shop) error
	FindByID(ctx context.Context, id string) (*Shop, error)
	List(ctx context.Context, offset int, limit int) ([]*Shop, int64, error)
	Delete(ctx context.Context, id string) error

	FindUserByPhone(ctx context.Context, phone string) (User, error)
	UserRegister(ctx context.Context, data *User) error
	GetUser(ctx context.Context, id int64) (User, error)
	CreateArticle(ctx context.Context, article *Article) error
	UpdateArticle(ctx context.Context, article *Article) error // 事务更新文章
	DeleteArticle(ctx context.Context, id int64) error      // 事务删除文章
	GetArticle(ctx context.Context, id int64) (Article, error)
	ListArticles(ctx context.Context, page int32, pageSize int32) ([]Article, int64, error)

	GetProduct(ctx context.Context, id int64) (Product, error) // 查询商品
	AddToCart(ctx context.Context, cart *Cart) error           // 加入购物车
	CreateOrder(ctx context.Context, order *Order) error       // 创建订单
	CreateComment(ctx context.Context, comment *Comment) error // 创建评论（事务：插入评论 + 文章评论数+1）
	ListCommentsByArticle(ctx context.Context, articleId int64) ([]Comment, error) // 查询文章评论列表
	UpdateArticleStatus(ctx context.Context, articleId int64, status int64) error // 更新文章审核状态
}
