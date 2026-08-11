package router

import (
	"github.com/gin-gonic/gin"

	"github.com/BwCloudWeGo/bw-cli/internal/gateway/handler"
)

// registerShopRoutes 在独立业务文件中注册 /api/v1/shops 接口。
func registerShopRoutes(v1 *gin.RouterGroup, shopHandler *handler.ShopHandler) {
	routes := v1.Group("/shops")
	routes.POST("", shopHandler.Create)
	routes.GET("", shopHandler.List)
	routes.GET("/:id", shopHandler.Get)
	routes.PUT("/:id", shopHandler.Update)
	routes.DELETE("/:id", shopHandler.Delete)
	//================================================================================================
	//短信发送接口
	routes.POST("/seed/sms", shopHandler.SeedSms)
	// 用户
	routes.POST("/register", shopHandler.Register)
	routes.POST("/login", shopHandler.Login)
	routes.GET("/user/:id", shopHandler.GetUser)
	// 文章
	routes.POST("/articles", shopHandler.CreateArticle)
	routes.GET("/articles/search", shopHandler.SearchArticles)
	routes.GET("/articles", shopHandler.ListArticles)
	routes.GET("/articles/:id", shopHandler.GetArticle)
	routes.PUT("/articles/:id", shopHandler.UpdateArticle)
	routes.DELETE("/articles/:id", shopHandler.DeleteArticle)
	// 购物车 & 下单
	routes.POST("/cart", shopHandler.AddToCart)
	routes.POST("/orders", shopHandler.CreateOrder)
	// 评论
	routes.POST("/comments", shopHandler.CreateComment)
	routes.GET("/articles/detail", shopHandler.GetArticleDetail)
	//支付宝异步回调
	routes.POST("/alipay", shopHandler.AlipayNotify)
}
