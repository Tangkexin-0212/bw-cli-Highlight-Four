package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"

	"github.com/BwCloudWeGo/bw-cli/pkg/grpcx"
	"github.com/BwCloudWeGo/bw-cli/pkg/httpx"
)

// outgoingContext forwards gateway metadata such as request id to downstream gRPC calls.
func outgoingContext(c *gin.Context) context.Context {
	return metadata.AppendToOutgoingContext(c.Request.Context(), grpcx.MetadataRequestID, httpx.RequestID(c))
}

// parseInt64Param 从路径参数中解析 int64，解析失败返回 0。
func parseInt64Param(c *gin.Context, key string) int64 {
	v, _ := strconv.ParseInt(c.Param(key), 10, 64)
	return v
}
