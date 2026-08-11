package router

import (
	"github.com/gin-gonic/gin"

	"github.com/BwCloudWeGo/bw-cli/internal/gateway/handler"
)

// registerCommentRoutes 在独立业务文件中注册 /api/v1/comments 接口。
func registerCommentRoutes(v1 *gin.RouterGroup, commentHandler *handler.CommentHandler) {
	routes := v1.Group("/comments")
	//routes.POST("", commentHandler.Create)
	routes.GET("", commentHandler.List)
	routes.GET("/:id", commentHandler.Get)
	routes.PUT("/:id", commentHandler.Update)
	routes.DELETE("/:id", commentHandler.Delete)
}
