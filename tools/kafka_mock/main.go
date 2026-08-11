package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/BwCloudWeGo/bw-cli/internal/shop/entity"
	"github.com/BwCloudWeGo/bw-cli/pkg/kafkax"
)

// 用法: go run ./tools/kafka_mock/main.go <文章ID>
// 功能: 向 Kafka 发送一条包含违规内容的模拟文章消息，用于测试审核不通过时 status 是否更新为 5
func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run ./tools/kafka_mock/main.go <文章ID>")
		fmt.Println("示例: go run ./tools/kafka_mock/main.go 1091112418219659264")
		os.Exit(1)
	}

	articleId, err := strconv.ParseInt(os.Args[1], 10, 64)
	if err != nil {
		fmt.Println("文章ID必须是数字:", err)
		os.Exit(1)
	}

	// 构造模拟文章（包含违规关键词，百度云审核会判定为不合规）
	now := time.Now()
	article := &entity.Article{
		Id:         articleId,
		UserId:     1090728432129544192,
		Title:      "测试审核违规内容",
		Content:    "赌博色情暴力毒品，这是一条用于测试内容审核的违规消息",
		Status:     0,
		Type:       1,
		CreatedAt:  &now,
	}

	// 序列化为 JSON（与 CreateArticle 中发送到 Kafka 的格式一致）
	data, err := json.Marshal(article)
	if err != nil {
		fmt.Println("JSON序列化失败:", err)
		os.Exit(1)
	}

	// 发送到 Kafka（使用 config.yaml 中的 broker 和 topic）
	cfg := kafkax.Config{
		Brokers: []string{"115.191.16.169:9092"},
		Topic:   "xiaolanshu-events",
	}
	producer, err := kafkax.NewProducer(cfg)
	if err != nil {
		fmt.Println("创建Kafka生产者失败:", err)
		os.Exit(1)
	}
	defer producer.Close()

	if err := producer.Publish(context.Background(), kafkax.Message{
		Value: data,
	}); err != nil {
		fmt.Println("发送Kafka消息失败:", err)
		os.Exit(1)
	}

	fmt.Printf("已发送模拟审核不通过的Kafka消息\n")
	fmt.Printf("  文章ID: %d\n", articleId)
	fmt.Printf("  标题: %s\n", article.Title)
	fmt.Printf("  内容: %s\n", article.Content)
	fmt.Println()
	fmt.Println("请观察shop服务日志，确认文章status是否更新为5（审核不通过）")
	fmt.Printf("可执行SQL验证: SELECT id, status FROM articles WHERE id = %d\n", articleId)
}
