package entity

import (
	"errors"

	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrCommentNotFound = errors.New("comment not found")
	ErrInvalidComment  = errors.New("invalid comment")
)

// Comment 是 comment 业务服务的聚合根。
// 业务明确后，请将 Name 和 Description 替换为真实业务字段。
type Comment struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewComment 校验输入，并创建带框架管理身份字段的聚合。
func NewComment(name string, description string) (*Comment, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" {
		return nil, ErrInvalidComment
	}
	now := time.Now().UTC()
	return &Comment{
		ID:          uuid.NewString(),
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Update 修改可变字段，并把校验保留在业务实体内部。
func (item *Comment) Update(name string, description string) error {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if item == nil || item.ID == "" || name == "" {
		return ErrInvalidComment
	}
	item.Name = name
	item.Description = description
	item.UpdatedAt = time.Now().UTC()
	return nil
}

//============================================================================================

type User struct { //用户表
	Id         int64  `gorm:"primaryKey"`        //雪花算法生成的全局唯一ID
	Phone      string `gorm:"type:varchar(11)"`  //手机号
	Password   string `gorm:"type:varchar(100)"` //密码(bcrypt加密)
	NickName   string `gorm:"type:varchar(50)"`  //昵称
	Avatar     string `gorm:"type:varchar(255)"` //头像URL
	Gender     int64  `gorm:"type:int(1)"`       //性别 0未知 1男 2女
	Status     int64  `gorm:"type:int(1)"`       //状态 1正常 2禁用 3注销
	ClientType string `gorm:"type:varchar(10)"`  //客户端类型 app/web
	CreatedAt  *time.Time
}

type Order struct { //订单表
	Id          int64      `gorm:"primaryKey"`         //雪花算法生成的全局唯一ID
	Name        string     `gorm:"type:varchar(100)"`  //订单名称（商品名称）
	OrderSn     int64      `gorm:"type:bigint"`        //订单号(冗余,便于查询)
	UserId      int64      `gorm:"type:bigint"`        //用户id(雪花ID)
	ShopId      int64      `gorm:"type:bigint"`        //商品id(雪花ID)
	Num         int64      `gorm:"type:int(11)"`       //购买数量
	TotalAmount float64    `gorm:"type:decimal(10,2)"` //订单总价
	PayAmount   float64    `gorm:"type:decimal(10,2)"` //实付金额
	PayMethod   int64      `gorm:"type:int(1)"`        //支付方式 1支付宝 2微信 3余额
	Status      int64      `gorm:"type:int(1)"`        //订单状态 1待支付 2已支付 3已发货 4已完成 5已取消 6已退款
	Address     string     `gorm:"type:varchar(255)"`  //收货地址
	PayTime     *time.Time //支付时间
	CreatedAt   *time.Time
}

func (o *Order) UpdateOrder(db *gorm.DB, no interface{}) {
	db.Debug().Model(Order{}).Where("id = ?", no).Update("status", 2)
}

type Product struct { //商品表
	Id          int64   `gorm:"primaryKey"`         //雪花算法生成的全局唯一ID
	UserId      int64   `gorm:"type:bigint"`        //商家id(雪花ID)
	Name        string  `gorm:"type:varchar(100)"`  //商品名称
	Description string  `gorm:"type:text"`          //商品描述
	Cover       string  `gorm:"type:varchar(255)"`  //商品封面URL
	Price       float64 `gorm:"type:decimal(10,2)"` //商品价格
	Stock       int64   `gorm:"type:int(11)"`       //库存数量
	SalesNum    int64   `gorm:"type:int(11)"`       //销量
	ViewNum     int64   `gorm:"type:int(11)"`       //浏览量
	CategoryId  int64   `gorm:"type:bigint"`        //分类id(雪花ID)
	Type        int64   `gorm:"type:int(1)"`        //商品类型 1普通商品 2热门商品 3推荐商品 4限时秒杀
	Status      int64   `gorm:"type:int(1)"`        //商品状态 1上架 2下架 3售罄 4审核中 5审核不通过
	CreatedAt   *time.Time
}

type Article struct { //文章表
	Id         int64  `gorm:"primaryKey"`        //雪花算法生成的全局唯一ID
	UserId     int64  `gorm:"type:bigint"`       //作者id(雪花ID)
	Title      string `gorm:"type:varchar(200)"` //文章标题
	Content    string `gorm:"type:longtext"`     //文章内容
	Cover      string `gorm:"type:varchar(255)"` //文章封面URL
	ViewNum    int64  `gorm:"type:int(11)"`      //阅读数
	LikeNum    int64  `gorm:"type:int(11)"`      //点赞数
	CommentNum int64  `gorm:"type:int(11)"`      //评论数
	CollectNum int64  `gorm:"type:int(11)"`      //收藏数
	ShopId     int64  `gorm:"type:bigint"`       //关联商品id(雪花ID)
	Type       int64  `gorm:"type:int(1)"`       //文章类型 1普通文章 2精华文章 3置顶文章 4专栏文章 5公告文章
	Status     int64  `gorm:"type:int(1)"`       //文章状态 1待审核 2审核中 3发布成功 4下架过 5审核不通过
	CreatedAt  *time.Time
}

type Cart struct { //用户购物车表
	Id        int64      `gorm:"primaryKey"`                //雪花算法生成的全局唯一ID
	UserId    int64      `gorm:"type:bigint"`               //用户id(雪花ID)
	ProductId int64      `gorm:"type:bigint"`               //商品id(雪花ID)
	Num       int64      `gorm:"type:int(11)"`              //购买数量
	Price     float64    `gorm:"type:decimal(10,2)"`        //加入时商品单价
	Selected  bool       `gorm:"type:tinyint(1);default:1"` //是否选中 1选中 0未选中
	CreatedAt *time.Time //加入时间
	UpdatedAt *time.Time //更新时间
}

type CommentNew struct { //评论表
	Id        int64      `gorm:"primaryKey"`  //雪花算法生成的全局唯一ID
	UserId    int64      `gorm:"type:bigint"` //评论用户id(雪花ID)
	ArticleId int64      `gorm:"type:bigint"` //文章id(雪花ID)
	Content   string     `gorm:"type:text"`   //评论内容
	CreatedAt *time.Time //评论时间
}
