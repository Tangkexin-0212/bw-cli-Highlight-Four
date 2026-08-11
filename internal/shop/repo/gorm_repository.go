package repo

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/BwCloudWeGo/bw-cli/internal/shop/entity"
	dbmodel "github.com/BwCloudWeGo/bw-cli/internal/shop/model"
)

// GormRepository 使用 Gorm 持久化 shop 聚合。
type GormRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

// NewGormRepository 创建 shop 仓储，并支持可选结构化日志。
func NewGormRepository(db *gorm.DB, loggers ...*zap.Logger) *GormRepository {
	log := zap.NewNop()
	if len(loggers) > 0 && loggers[0] != nil {
		log = loggers[0]
	}
	return &GormRepository{db: db, log: log}
}

// AutoMigrate 创建或更新 shops 表结构。
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&dbmodel.ShopModel{},
		&entity.Product{},
		&entity.Article{},
		&entity.Order{},
		&entity.User{},
		&entity.Cart{},
		&entity.Comment{},
	)
}

// Save 新增或更新 shop 聚合。
func (r *GormRepository) Save(ctx context.Context, item *entity.Shop) error {
	start := time.Now()
	tx := r.db.WithContext(ctx).Save(toRecord(item))
	r.logOperation("Save", tx.RowsAffected, start, tx.Error)
	return tx.Error
}

// FindByID 根据 ID 加载 shop 聚合。
func (r *GormRepository) FindByID(ctx context.Context, id string) (*entity.Shop, error) {
	start := time.Now()
	var record dbmodel.ShopModel
	tx := r.db.WithContext(ctx).Where("id = ?", id).First(&record)
	err := tx.Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = entity.ErrShopNotFound
	}
	if err != nil {
		r.logOperation("FindByID", tx.RowsAffected, start, err)
		return nil, err
	}
	r.logOperation("FindByID", tx.RowsAffected, start, nil)
	return toDomain(&record), nil
}

// List 加载分页 shop 聚合。
func (r *GormRepository) List(ctx context.Context, offset int, limit int) ([]*entity.Shop, int64, error) {
	start := time.Now()
	var total int64
	countTx := r.db.WithContext(ctx).Model(&dbmodel.ShopModel{}).Count(&total)
	if countTx.Error != nil {
		r.logOperation("Count", countTx.RowsAffected, start, countTx.Error)
		return nil, 0, countTx.Error
	}
	var records []dbmodel.ShopModel
	tx := r.db.WithContext(ctx).
		Order("created_at desc").
		Offset(offset).
		Limit(limit).
		Find(&records)
	if tx.Error != nil {
		r.logOperation("List", tx.RowsAffected, start, tx.Error)
		return nil, 0, tx.Error
	}
	items := make([]*entity.Shop, 0, len(records))
	for i := range records {
		items = append(items, toDomain(&records[i]))
	}
	r.logOperation("List", tx.RowsAffected, start, nil)
	return items, total, nil
}

// Delete 根据 ID 删除 shop 聚合。
func (r *GormRepository) Delete(ctx context.Context, id string) error {
	start := time.Now()
	tx := r.db.WithContext(ctx).Where("id = ?", id).Delete(&dbmodel.ShopModel{})
	err := tx.Error
	if err == nil && tx.RowsAffected == 0 {
		err = entity.ErrShopNotFound
	}
	r.logOperation("Delete", tx.RowsAffected, start, err)
	return err
}

func (r *GormRepository) logOperation(operation string, rows int64, start time.Time, err error) {
	fields := []zap.Field{
		zap.String("repository", "shop"),
		zap.String("operation", operation),
		zap.Int64("rows_affected", rows),
		zap.Float64("latency_ms", float64(time.Since(start).Microseconds())/1000),
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
		r.log.Warn("repository operation completed with error", fields...)
		return
	}
	r.log.Info("repository operation completed", fields...)
}

func toRecord(item *entity.Shop) *dbmodel.ShopModel {
	return &dbmodel.ShopModel{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func toDomain(record *dbmodel.ShopModel) *entity.Shop {
	return &entity.Shop{
		ID:          record.ID,
		Name:        record.Name,
		Description: record.Description,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}

var _ entity.Repository = (*GormRepository)(nil)

//=====================================================================

func (r *GormRepository) FindUserByPhone(ctx context.Context, phone string) (entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error
	return user, err
}

func (r *GormRepository) UserRegister(ctx context.Context, data *entity.User) error {
	err := r.db.WithContext(ctx).Create(&data).Error
	return err
}

func (r *GormRepository) GetUser(ctx context.Context, id int64) (entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	return user, err
}

func (r *GormRepository) CreateArticle(ctx context.Context, article *entity.Article) error {
	err := r.db.WithContext(ctx).Create(&article).Error
	return err
}

// UpdateArticle 使用事务更新文章，保证数据一致性
func (r *GormRepository) UpdateArticle(ctx context.Context, article *entity.Article) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Model(&entity.Article{}).Where("id = ?", article.Id).Updates(article).Error
	})
}

// DeleteArticle 使用事务删除文章
func (r *GormRepository) DeleteArticle(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Where("id = ?", id).Delete(&entity.Article{}).Error
	})
}

func (r *GormRepository) GetArticle(ctx context.Context, id int64) (entity.Article, error) {
	var article entity.Article
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&article).Error
	return article, err
}

func (r *GormRepository) ListArticles(ctx context.Context, page int32, pageSize int32) ([]entity.Article, int64, error) {
	var articles []entity.Article
	var total int64
	offset := int((page - 1) * pageSize)
	r.db.WithContext(ctx).Model(&entity.Article{}).Count(&total)
	err := r.db.WithContext(ctx).Order("id desc").Offset(offset).Limit(int(pageSize)).Find(&articles).Error
	return articles, total, err
}

// GetProduct 根据ID查询商品
func (r *GormRepository) GetProduct(ctx context.Context, id int64) (entity.Product, error) {
	var product entity.Product
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&product).Error
	return product, err
}

// AddToCart 将商品加入购物车
func (r *GormRepository) AddToCart(ctx context.Context, cart *entity.Cart) error {
	return r.db.WithContext(ctx).Create(cart).Error
}

// CreateOrder 创建订单
func (r *GormRepository) CreateOrder(ctx context.Context, order *entity.Order) error {
	return r.db.WithContext(ctx).Create(order).Error
}

// CreateComment 事务：插入评论 + 文章评论数自增
func (r *GormRepository) CreateComment(ctx context.Context, comment *entity.Comment) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(comment).Error; err != nil {
			return err
		}
		return tx.Model(&entity.Article{}).Where("id = ?", comment.ArticleId).
			UpdateColumn("comment_num", gorm.Expr("comment_num + 1")).Error
	})
}

// ListCommentsByArticle 查询文章下的所有评论，按时间倒序
func (r *GormRepository) ListCommentsByArticle(ctx context.Context, articleId int64) ([]entity.Comment, error) {
	var comments []entity.Comment
	err := r.db.WithContext(ctx).Where("article_id = ?", articleId).
		Order("created_at desc").Find(&comments).Error
	return comments, err
}

// UpdateArticleStatus 更新文章审核状态
func (r *GormRepository) UpdateArticleStatus(ctx context.Context, articleId int64, status int64) error {
	return r.db.WithContext(ctx).Model(&entity.Article{}).
		Where("id = ?", articleId).Update("status", status).Error
}
