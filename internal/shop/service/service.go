package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/BwCloudWeGo/bw-cli/pkg/alipayx"
	"github.com/BwCloudWeGo/bw-cli/pkg/esx"
	"github.com/BwCloudWeGo/bw-cli/pkg/kafkax"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"gorm.io/gorm"

	shopv1 "github.com/BwCloudWeGo/bw-cli/api/gen/shop/v1"
	"github.com/BwCloudWeGo/bw-cli/internal/shop/dto"
	"github.com/BwCloudWeGo/bw-cli/internal/shop/entity"
	"github.com/BwCloudWeGo/bw-cli/pkg/config"
	"github.com/BwCloudWeGo/bw-cli/pkg/redisx"
	"github.com/BwCloudWeGo/bw-cli/pkg/utils"
)

// Service 编排 shop 用例。
type Service struct {
	repo   entity.Repository
	alipay *alipayx.Client
	log    *zap.Logger
}

// NewService 创建 shop 用例服务。
func NewService(repo entity.Repository, alipay *alipayx.Client, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{repo: repo, alipay: alipay, log: log}
}

// Create 创建 shop 记录。
func (s *Service) Create(ctx context.Context, cmd dto.CreateCommand) (*dto.ShopDTO, error) {
	item, err := entity.NewShop(cmd.Name, cmd.Description)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, item); err != nil {
		return nil, err
	}
	s.log.Info("shop created", zap.String("aggregate_id", item.ID), zap.String("use_case", "CreateShop"))
	return dto.FromShop(item), nil
}

// Get 根据 ID 返回一条 shop 记录。
func (s *Service) Get(ctx context.Context, id string) (*dto.ShopDTO, error) {
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.FromShop(item), nil
}

// List 返回分页的 shop 记录。
func (s *Service) List(ctx context.Context, cmd dto.ListCommand) (*dto.ListShopDTO, error) {
	offset, limit := normalizePagination(cmd.Page, cmd.PageSize)
	items, total, err := s.repo.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	output := &dto.ListShopDTO{Items: make([]*dto.ShopDTO, 0, len(items)), Total: total}
	for _, item := range items {
		output.Items = append(output.Items, dto.FromShop(item))
	}
	return output, nil
}

// Update 根据 ID 修改一条 shop 记录。
func (s *Service) Update(ctx context.Context, cmd dto.UpdateCommand) (*dto.ShopDTO, error) {
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
	s.log.Info("shop updated", zap.String("aggregate_id", cmd.ID), zap.String("use_case", "UpdateShop"))
	return dto.FromShop(item), nil
}

// Delete 根据 ID 删除一条 shop 记录。
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.log.Info("shop deleted", zap.String("aggregate_id", id), zap.String("use_case", "DeleteShop"))
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

//=====================================================================

func (s *Service) SeedSms(ctx context.Context, req *shopv1.SeedSmsRequest) (*shopv1.SeedSmsResponse, error) {
	rdb := redisx.NewClient(config.GlobalConfig.Redis)
	Key := fmt.Sprintf("seed_sms_%s", req.Phone)
	Key1 := fmt.Sprintf("seed_code_%s", req.Phone)
	Code := rand.Intn(9000) + 1000
	rdb.Set(ctx, Key1, Code, time.Minute*1)
	//调用短信平台包
	sms, err := utils.SendSms(strconv.Itoa(Code), req.Phone)
	if err != nil {
		return nil, err
	}
	if sms.Code != 2 {
		return nil, errors.New(sms.Msg)
	}
	//单个手机号一分钟只能发送三次
	count, err := rdb.Get(ctx, Key).Int64()
	if count >= 3 {
		fmt.Println("单个手机号一分钟只能发送三次")
		return nil, err
	}
	//手机号封禁1分钟
	val := rdb.Incr(ctx, Key).Val()
	if val == 1 {
		rdb.Expire(ctx, Key, time.Minute*1)
	}

	return &shopv1.SeedSmsResponse{
		Code: int64(Code),
	}, nil
}

func (s *Service) Register(ctx context.Context, req *shopv1.RegisterRequest) (*shopv1.RegisterResponse, error) {
	//1.查询数据库中手机号是否已存在
	user, err := s.repo.FindUserByPhone(ctx, req.Phone)
	// 关键修复: gorm.ErrRecordNotFound 表示手机号未注册, 是合法情况, 应继续走注册流程
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		fmt.Println("查询手机号失败")
		return nil, err
	}
	if user.Id != 0 {
		fmt.Println("手机号已存在，单个手机号只能注册一次")
		return nil, errors.New("手机号已存在")
	}

	//2.查询验证码是否一致
	rdb := redisx.NewClient(config.GlobalConfig.Redis)
	Key := fmt.Sprintf("seed_code_%s", req.Phone)
	code, err := rdb.Get(ctx, Key).Result()
	if err != nil || code != req.Code {
		fmt.Println("验证码错误,注册失败")
		return nil, errors.New("验证码错误")
	}

	//3.将数据添加到数据库中
	ID := utils.GetSnowId()
	Time := time.Now()
	data := &entity.User{
		Id:         ID,
		Phone:      req.Phone,
		Password:   utils.Md5Str(req.Password),
		NickName:   req.Nickname,
		Avatar:     "https://www.bing.com/th/id/OIP.WiP9RA-W2KGc-hFAd0tD1wAAAA?w=193&h=193&c=8&rs=1&qlt=90&o=6&dpr=1.8&pid=ImgAns&rm=2", //系统默认头像
		Gender:     0,                                                                                                                   //默认性别未知
		Status:     1,                                                                                                                   //默认状态为正常
		ClientType: req.ClientType,
		CreatedAt:  &Time,
	}
	if err = s.repo.UserRegister(ctx, data); err != nil {
		fmt.Println("用户注册失败")
		return nil, err
	}

	return &shopv1.RegisterResponse{
		UserId: data.Id,
	}, nil
}

func (s *Service) Login(ctx context.Context, req *shopv1.LoginRequest) (*shopv1.LoginResponse, error) {
	//1.查询数据库中手机号是否已存在
	user, err := s.repo.FindUserByPhone(ctx, req.Phone)
	if err != nil {
		fmt.Println("查询手机号失败")
		return nil, err
	}
	if user.Id == 0 {
		fmt.Println("手机号不存在")
		return nil, errors.New("手机号不存在")
	}
	//2.验证密码是否正确,错误三次设置一小时过期时间
	rdb := redisx.NewClient(config.GlobalConfig.Redis)
	key := fmt.Sprintf("login_err_%v", req.Phone)

	//3.检查是否已被封禁
	count, err := rdb.Get(ctx, key).Int64()
	if count >= 3 {
		fmt.Println("账号因多次密码错误被临时封禁，请1小时后重试")
		return nil, err
	}

	//4.验证密码
	if user.Password != utils.Md5Str(req.Password) {
		val := rdb.Incr(ctx, key).Val()
		if val >= 3 {
			rdb.Expire(ctx, key, time.Hour*1) // 封禁1小时
			return nil, errors.New("密码错误已达3次，账号封禁1小时")
		}
		return nil, errors.New("密码错误，请重试")
	}

	//5.密码正确，重置计数
	rdb.Del(ctx, key)

	return &shopv1.LoginResponse{
		UserId: user.Id,
	}, nil
}

func (s *Service) GetUser(ctx context.Context, req *shopv1.GetUserRequest) (*shopv1.UserResponse, error) {

	user, err := s.repo.GetUser(ctx, req.Id)
	if err != nil {
		fmt.Println("查询用户失败")
		return nil, err
	}

	return &shopv1.UserResponse{
		Id:       user.Id,
		Phone:    user.Phone,
		Nickname: user.NickName,
		Avatar:   user.Avatar,
		Gender:   user.Gender,
		Status:   user.Status,
	}, nil
}

func (s *Service) CreateArticle(ctx context.Context, req *shopv1.CreateArticleRequest) (*shopv1.ArticleResponse, error) {
	//1.查询作者是否存在
	_, err := s.repo.GetUser(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	//2.发布文章
	CreatedAt := time.Now() // 获取当前时间
	ID := utils.GetSnowId() // 雪花算法随机生成id

	article := &entity.Article{
		Id:         ID,
		UserId:     req.UserId,
		Title:      req.Title,
		Content:    req.Content,
		Cover:      req.Cover,
		ViewNum:    0,
		LikeNum:    0,
		CommentNum: 0,
		CollectNum: 0,
		ShopId:     req.ShopId,
		Type:       req.Type,
		Status:     0,
		CreatedAt:  &CreatedAt,
	}
	err = s.repo.CreateArticle(ctx, article)
	if err != nil {
		fmt.Println("文章发布失败")
		return nil, err
	}

	// *time.Time 转 string, 防止 nil 指针 panic
	var createdAt string
	if article.CreatedAt != nil {
		createdAt = article.CreatedAt.Format("2006-01-02 15:04:05")
	}

	//将文章写入kafka中
	articleMarshal, err := json.Marshal(article)
	if err != nil {
		return nil, err
	}
	producer, err := kafkax.NewProducer(config.GlobalConfig.Kafka)
	if err != nil {
		return nil, err
	}
	err = producer.Publish(ctx, kafkax.Message{
		Value: articleMarshal,
	})
	if err != nil {
		return nil, err
	}

	//3.同步到es中
	EsArticle := map[string]interface{}{
		"ArticleId":  article.Id,
		"UserId":     req.UserId,
		"Title":      req.Title,
		"Content":    req.Content,
		"Cover":      req.Cover,
		"ViewNum":    article.ViewNum,
		"LikeNum":    article.LikeNum,
		"CommentNum": article.CommentNum,
		"CollectNum": article.CollectNum,
		"ShopId":     req.ShopId,
		"Type":       req.Type,
		"CreatedAt":  createdAt,
	}
	marshal, err := json.Marshal(EsArticle)
	if err != nil {
		return nil, err
	}
	es, _ := esx.NewClient(config.GlobalConfig.Elasticsearch)
	EsReq := esapi.IndexRequest{
		Index: "article",
		Body:  bytes.NewReader(marshal),
	}

	// Perform the request with the client.
	_, err = EsReq.Do(context.Background(), es)
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}

	return &shopv1.ArticleResponse{
		Id:         article.Id,
		UserId:     article.UserId,
		Title:      article.Title,
		Content:    article.Content,
		Cover:      article.Cover,
		ViewNum:    article.ViewNum,
		LikeNum:    article.LikeNum,
		CommentNum: article.CommentNum,
		CollectNum: article.CollectNum,
		ShopId:     article.ShopId,
		Type:       article.Type,
		Status:     article.Status,
		CreatedAt:  createdAt,
	}, nil
}

func (s *Service) KafkaQueue(ctx context.Context) {
	consumer, err := kafkax.NewConsumer(config.GlobalConfig.Kafka)
	if err != nil {
		return
	}
	err = consumer.Run(ctx, func(ctx context.Context, message kafka.Message) error {
		var article *entity.Article
		if err := json.Unmarshal(message.Value, &article); err != nil {
			return nil // 单条消息解析失败，跳过继续消费
		}
		pass := utils.BaiduPass(article.Title + article.Content)
		if pass == 1 {
			// 审核通过，文章状态改为发布成功
			err := s.repo.UpdateArticleStatus(ctx, article.Id, 3)
			if err != nil {
				return err
			}
		} else {
			// 审核不通过，文章状态改为审核不通过
			err := s.repo.UpdateArticleStatus(ctx, article.Id, 5)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return
	}
}

func (s *Service) GetArticle(ctx context.Context, req *shopv1.GetArticleRequest) (*shopv1.ArticleResponse, error) {
	article, err := s.repo.GetArticle(ctx, req.Id)
	if err != nil {
		fmt.Println("文章查询失败")
		return nil, err
	}

	return &shopv1.ArticleResponse{
		Id:         article.Id,
		UserId:     article.UserId,
		Title:      article.Title,
		Content:    article.Content,
		Cover:      article.Cover,
		ViewNum:    article.ViewNum,
		LikeNum:    article.LikeNum,
		CommentNum: article.CommentNum,
		CollectNum: article.CollectNum,
		ShopId:     article.ShopId,
		Type:       article.Type,
		Status:     article.Status,
		CreatedAt:  article.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (s *Service) ListArticles(ctx context.Context, req *shopv1.ListArticlesRequest) (*shopv1.ListArticlesResponse, error) {
	//1.分页查询文章列表
	var List []*shopv1.ArticleResponse

	articles, total, err := s.repo.ListArticles(ctx, req.Page, req.PageSize)
	if err != nil {
		fmt.Println("文章列表查询失败")
		return nil, err
	}

	for _, article := range articles {
		var createdAt string
		if article.CreatedAt != nil {
			createdAt = article.CreatedAt.Format("2006-01-02 15:04:05")
		}

		t := &shopv1.ArticleResponse{
			Id:         article.Id,
			UserId:     article.UserId,
			Title:      article.Title,
			Content:    article.Content,
			Cover:      article.Cover,
			ViewNum:    article.ViewNum,
			LikeNum:    article.LikeNum,
			CommentNum: article.CommentNum,
			CollectNum: article.CollectNum,
			ShopId:     article.ShopId,
			Type:       article.Type,
			Status:     article.Status,
			CreatedAt:  createdAt,
		}

		List = append(List, t)
	}

	return &shopv1.ListArticlesResponse{
		Items: List,
		Total: total,
	}, nil
}

func (s *Service) UpdateArticle(ctx context.Context, req *shopv1.UpdateArticleRequest) (*shopv1.ArticleResponse, error) {
	// 1. 查询文章是否存在
	article, err := s.repo.GetArticle(ctx, req.Id)
	if err != nil {
		fmt.Println("文章查询失败")
		return nil, err
	}

	// 2. 更新可变字段（只更新请求中传入的字段）
	if req.Title != "" {
		article.Title = req.Title
	}
	if req.Content != "" {
		article.Content = req.Content
	}
	if req.Cover != "" {
		article.Cover = req.Cover
	}
	if req.Type != 0 {
		article.Type = req.Type
	}
	if req.Status != 0 {
		article.Status = req.Status
	}

	// 3. 事务更新文章
	if err := s.repo.UpdateArticle(ctx, &article); err != nil {
		fmt.Println("文章更新失败")
		return nil, err
	}

	// 4. 格式化时间，同步更新 ES
	var createdAt string
	if article.CreatedAt != nil {
		createdAt = article.CreatedAt.Format("2006-01-02 15:04:05")
	}

	esDoc := map[string]interface{}{
		"ArticleId":  article.Id,
		"UserId":     article.UserId,
		"Title":      article.Title,
		"Content":    article.Content,
		"Cover":      article.Cover,
		"ViewNum":    article.ViewNum,
		"LikeNum":    article.LikeNum,
		"CommentNum": article.CommentNum,
		"CollectNum": article.CollectNum,
		"ShopId":     article.ShopId,
		"Type":       article.Type,
		"Status":     article.Status,
		"CreatedAt":  createdAt,
	}
	marshal, err := json.Marshal(esDoc)
	if err != nil {
		return nil, err
	}
	es, _ := esx.NewClient(config.GlobalConfig.Elasticsearch)
	esReq := esapi.IndexRequest{
		Index:      "article",
		DocumentID: strconv.FormatInt(article.Id, 10), // 指定文档ID，覆盖更新而非新增
		Body:       bytes.NewReader(marshal),
	}
	_, err = esReq.Do(context.Background(), es)
	if err != nil {
		log.Fatalf("ES同步失败: %s", err)
	}

	return &shopv1.ArticleResponse{
		Id:         article.Id,
		UserId:     article.UserId,
		Title:      article.Title,
		Content:    article.Content,
		Cover:      article.Cover,
		ViewNum:    article.ViewNum,
		LikeNum:    article.LikeNum,
		CommentNum: article.CommentNum,
		CollectNum: article.CollectNum,
		ShopId:     article.ShopId,
		Type:       article.Type,
		Status:     article.Status,
		CreatedAt:  createdAt,
	}, nil
}

func (s *Service) DeleteArticle(ctx context.Context, req *shopv1.DeleteArticleRequest) (*shopv1.DeleteArticleResponse, error) {
	// 1. 确认文章存在
	if _, err := s.repo.GetArticle(ctx, req.Id); err != nil {
		fmt.Println("文章不存在")
		return nil, err
	}

	// 2. 事务删除文章
	if err := s.repo.DeleteArticle(ctx, req.Id); err != nil {
		fmt.Println("文章删除失败")
		return nil, err
	}

	// 3. 同步删除 ES 中的文档
	es, _ := esx.NewClient(config.GlobalConfig.Elasticsearch)
	esReq := esapi.DeleteRequest{
		Index:      "article",
		DocumentID: strconv.FormatInt(req.Id, 10),
	}
	_, err := esReq.Do(context.Background(), es)
	if err != nil {
		log.Fatalf("ES删除失败: %s", err)
	}

	return &shopv1.DeleteArticleResponse{
		Success: true,
	}, nil
}

func (s *Service) SearchArticles(ctx context.Context, req *shopv1.SearchArticlesRequest) (*shopv1.ListArticlesResponse, error) {
	//1.初始化es
	es, _ := esx.NewClient(config.GlobalConfig.Elasticsearch)
	searcher := esx.NewSearcherFromClient(es)

	//2.拼接关键词，同时搜标题和正文
	Keyword := req.Title + req.Content
	if Keyword == "" {
		return &shopv1.ListArticlesResponse{}, nil
	}
	Fields := []string{"Title", "Content"}

	//3.es模糊搜索+点赞排序+高亮
	result, err := searcher.FuzzySearch(ctx, esx.FuzzySearchRequest{
		Index:   "article",
		Keyword: Keyword,
		Fields:  Fields,
		Sort:    []esx.Sort{esx.SortField("LikeNum", "desc")},
		From:    int((req.Page - 1) * req.PageSize),
		Size:    int(req.PageSize),
		Highlight: esx.HighlightConfig{
			Fields:            Fields,
			PreTags:           []string{"<span style='color: red'>"},
			PostTags:          []string{"</span>"},
			FragmentSize:      120,
			NumberOfFragments: 2,
		},
	})
	if err != nil {
		fmt.Println("Es搜索失败")
		return nil, err
	}

	//4.数据转换格式
	var List []*shopv1.ArticleResponse
	for _, hit := range result.Hits {
		var doc struct {
			ArticleId  int64  `json:"ArticleId"`
			UserId     int64  `json:"UserId"`
			Title      string `json:"Title"`
			Content    string `json:"Content"`
			Cover      string `json:"Cover"`
			ViewNum    int64  `json:"ViewNum"`
			LikeNum    int64  `json:"LikeNum"`
			CommentNum int64  `json:"CommentNum"`
			CollectNum int64  `json:"CollectNum"`
			ShopId     int64  `json:"ShopId"`
			Type       int64  `json:"Type"`
			Status     int64  `json:"Status"`
			CreatedAt  string `json:"CreatedAt"`
		}
		if err := json.Unmarshal(hit.Source, &doc); err != nil {
			return nil, err
		}
		if v, ok := hit.Highlight["Title"]; ok {
			doc.Title = v[0]
		}
		if v, ok := hit.Highlight["Content"]; ok {
			doc.Content = v[0]
		}
		List = append(List, &shopv1.ArticleResponse{
			Id:         doc.ArticleId,
			UserId:     doc.UserId,
			Title:      doc.Title,
			Content:    doc.Content,
			Cover:      doc.Cover,
			ViewNum:    doc.ViewNum,
			LikeNum:    doc.LikeNum,
			CommentNum: doc.CommentNum,
			CollectNum: doc.CollectNum,
			ShopId:     doc.ShopId,
			Type:       doc.Type,
			Status:     doc.Status,
			CreatedAt:  doc.CreatedAt,
		})
	}

	return &shopv1.ListArticlesResponse{
		Items: List,
		Total: result.Total,
	}, nil
}

// AddToCart 将商品加入购物车，校验用户和商品状态
func (s *Service) AddToCart(ctx context.Context, req *shopv1.AddToCartRequest) (*shopv1.AddToCartResponse, error) {
	// 1. 校验用户是否存在且状态正常（Status == 1）
	user, err := s.repo.GetUser(ctx, req.UserId)
	if err != nil {
		return nil, entity.ErrProductNotFound
	}
	if user.Status != 1 {
		return nil, entity.ErrUserNotAllowed
	}

	// 2. 校验商品是否存在且已上架（Status == 1）
	product, err := s.repo.GetProduct(ctx, req.ProductId)
	if err != nil {
		return nil, entity.ErrProductNotFound
	}
	if product.Status != 1 {
		return nil, entity.ErrProductNotOnSale
	}

	// 3. 写入购物车记录
	now := time.Now()
	cart := &entity.Cart{
		Id:        utils.GetSnowId(),
		UserId:    req.UserId,
		ProductId: req.ProductId,
		Num:       req.Num,
		Price:     product.Price,
		Selected:  true,
		CreatedAt: &now,
		UpdatedAt: &now,
	}
	if err := s.repo.AddToCart(ctx, cart); err != nil {
		return nil, err
	}

	return &shopv1.AddToCartResponse{
		Id: cart.Id,
	}, nil
}

// CreateOrder 下单：校验用户和商品 → 计算总价 → 创建订单 → 生成支付宝支付链接
func (s *Service) CreateOrder(ctx context.Context, req *shopv1.CreateOrderRequest) (*shopv1.CreateOrderResponse, error) {
	// 1. 校验用户
	user, err := s.repo.GetUser(ctx, req.UserId)
	if err != nil {
		return nil, entity.ErrProductNotFound
	}
	if user.Status != 1 {
		return nil, entity.ErrUserNotAllowed
	}

	// 2. 校验商品
	product, err := s.repo.GetProduct(ctx, req.ProductId)
	if err != nil {
		return nil, entity.ErrProductNotFound
	}
	if product.Status != 1 {
		return nil, entity.ErrProductNotOnSale
	}

	// 3. 计算总价 = 商品单价 × 数量
	totalAmount := product.Price * float64(req.Num)

	// 4. 创建订单
	now := time.Now()
	OrderSn := utils.GetSnowId()
	order := &entity.Order{
		Id:          utils.GetSnowId(),
		Name:        product.Name, // 订单名称 = 商品名
		OrderSn:     OrderSn,      // 生成唯一订单号
		UserId:      req.UserId,
		ShopId:      req.ProductId,
		Num:         req.Num,
		TotalAmount: totalAmount,
		PayAmount:   totalAmount,
		PayMethod:   req.PayMethod,
		Status:      1, // 待支付
		Address:     req.Address,
		CreatedAt:   &now,
	}
	if err := s.repo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}

	// 5. 生成支付宝支付链接
	alipay, err := alipayx.NewClient(config.GlobalConfig.Alipay)
	if err != nil {
		fmt.Println("初始化Alipay失败")
		return nil, err
	}

	url, err := alipay.WapPayURL(alipayx.PayRequest{
		OutTradeNo:  strconv.FormatInt(OrderSn, 10),
		Subject:     "小蓝书支付订单",
		TotalAmount: strconv.FormatFloat(totalAmount, 'f', 2, 64),
	})
	if err != nil {
		fmt.Println("订单支付链接生成失败")
		return nil, err
	}

	return &shopv1.CreateOrderResponse{
		OrderId: order.Id,
		OrderSn: order.OrderSn,
		PayUrl:  url,
	}, nil
}

// CreateComment 用户评论文章：验证用户和文章存在 → 事务写入评论并自增文章评论数
func (s *Service) CreateComment(ctx context.Context, req *shopv1.CreateCommentRequest) (*shopv1.CreateCommentResponse, error) {
	// 1. 校验用户是否存在
	if _, err := s.repo.GetUser(ctx, req.UserId); err != nil {
		return nil, err
	}
	// 2. 校验文章是否存在
	if _, err := s.repo.GetArticle(ctx, req.ArticleId); err != nil {
		return nil, err
	}
	// 3. 事务：插入评论 + 文章评论数+1
	now := time.Now()
	comment := &entity.Comment{
		Id:        utils.GetSnowId(),
		UserId:    req.UserId,
		ArticleId: req.ArticleId,
		Content:   req.Content,
		CreatedAt: &now,
	}
	if err := s.repo.CreateComment(ctx, comment); err != nil {
		return nil, err
	}
	return &shopv1.CreateCommentResponse{Id: comment.Id}, nil
}

// GetArticleDetail 文章聚合详情：返回文章信息 + 评论列表（含评论者昵称）
func (s *Service) GetArticleDetail(ctx context.Context, req *shopv1.GetArticleDetailRequest) (*shopv1.ArticleDetailResponse, error) {
	// 1. 查询文章
	article, err := s.repo.GetArticle(ctx, req.ArticleId)
	if err != nil {
		return nil, err
	}
	// 2. 查询评论列表
	comments, err := s.repo.ListCommentsByArticle(ctx, req.ArticleId)
	if err != nil {
		return nil, err
	}
	// 3. 组装评论（含用户昵称）
	commentList := make([]*shopv1.CommentResponse, 0, len(comments))
	for _, c := range comments {
		nickname := ""
		createdAt := ""
		if user, err := s.repo.GetUser(ctx, c.UserId); err == nil {
			nickname = user.NickName
		}
		if c.CreatedAt != nil {
			createdAt = c.CreatedAt.Format("2006-01-02 15:04:05")
		}
		commentList = append(commentList, &shopv1.CommentResponse{
			Id:        c.Id,
			UserId:    c.UserId,
			Nickname:  nickname,
			Content:   c.Content,
			CreatedAt: createdAt,
		})
	}
	// 4. 组装文章
	var articleCreatedAt string
	if article.CreatedAt != nil {
		articleCreatedAt = article.CreatedAt.Format("2006-01-02 15:04:05")
	}
	return &shopv1.ArticleDetailResponse{
		Article: &shopv1.ArticleResponse{
			Id:         article.Id,
			UserId:     article.UserId,
			Title:      article.Title,
			Content:    article.Content,
			Cover:      article.Cover,
			ViewNum:    article.ViewNum,
			LikeNum:    article.LikeNum,
			CommentNum: article.CommentNum,
			CollectNum: article.CollectNum,
			ShopId:     article.ShopId,
			Type:       article.Type,
			Status:     article.Status,
			CreatedAt:  articleCreatedAt,
		},
		Comments: commentList,
	}, nil
}
