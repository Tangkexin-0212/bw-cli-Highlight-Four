package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/BwCloudWeGo/bw-cli/pkg/nacosx"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	shopv1 "github.com/BwCloudWeGo/bw-cli/api/gen/shop/v1"
	shophandler "github.com/BwCloudWeGo/bw-cli/internal/shop/handler"
	shoprepo "github.com/BwCloudWeGo/bw-cli/internal/shop/repo"
	shopservice "github.com/BwCloudWeGo/bw-cli/internal/shop/service"
	"github.com/BwCloudWeGo/bw-cli/pkg/alipayx"
	"github.com/BwCloudWeGo/bw-cli/pkg/config"
	"github.com/BwCloudWeGo/bw-cli/pkg/database"
	"github.com/BwCloudWeGo/bw-cli/pkg/grpcx"
	"github.com/BwCloudWeGo/bw-cli/pkg/logger"
)

const serviceName = "shop-service"
const defaultGRPCPort = 9101

func main() {
	if err := config.InitGlobal("configs/config.yaml"); err != nil {
		panic(err)
	}
	cfg := config.MustGlobal()
	cfg.Log.Service = cfg.ServiceName("shop")
	cfg.Log = logger.WithDailyFileName(cfg.Log, time.Now())

	log, err := logger.New(cfg.Log)
	if err != nil {
		panic(err)
	}
	defer log.Sync()
	config.PrintSourceNotice(cfg, os.Stdout)

	db, err := database.Open(cfg.Database, cfg.MySQL, cfg.PostgreSQL, log)
	if err != nil {
		log.Fatal("open database failed", zap.Error(err))
	}

	if err := shoprepo.AutoMigrate(db); err != nil {
		log.Fatal("migrate shop database failed", zap.Error(err))
	}

	repo := shoprepo.NewGormRepository(db, log)
	payClient, err := alipayx.NewClient(cfg.Alipay)
	if err != nil {
		log.Warn("alipay client not initialized, payment features disabled", zap.Error(err))
	}
	svc := shopservice.NewService(repo, payClient, log)
	server := grpc.NewServer(grpc.UnaryInterceptor(grpcx.UnaryServerInterceptor(log)))
	shopv1.RegisterShopServiceServer(server, shophandler.NewServer(svc, log))

	port := cfg.ServicePort("shop", defaultGRPCPort)
	addr := fmt.Sprintf("%s:%d", cfg.GRPC.Host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n[Service Start Failed]\n  service: %s\n  listen: %s\n  error: %v\n\n", serviceName, addr, err)
		log.Fatal("listen failed", zap.String("addr", addr), zap.Error(err))
	}

	//=============================================================================

	//Nacos注册服务
	if cfg.Nacos.Enabled {
		service, err := nacosx.RegisterGRPCService(nacosx.WithDefaults(cfg.Nacos), "note", cfg.GRPC.Host, port)
		if err != nil {
			fmt.Println("nacos注册失败", service)
			return
		}
	}

	//启动kafka消费者队列
	go kafkaQueue(svc, log)

	//=============================================================================

	printStartupSummary(cfg.App.Env, addr, port)
	go shutdownOnSignal(server, log)
	if err := server.Serve(listener); err != nil {
		log.Fatal("service stopped unexpectedly", zap.Error(err))
	}
}

func kafkaQueue(svc *shopservice.Service, log *zap.Logger) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		log.Info("Kafka:正在停止...")
	}()
	log.Info("Kafka:已经启动...")
	svc.KafkaQueue(context.Background())
}

func printStartupSummary(env string, addr string, port int) {
	host := strings.Split(addr, ":")[0]
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	fmt.Fprintf(os.Stdout, "\n[Service Started]\n")
	fmt.Fprintf(os.Stdout, "  service: %s\n", serviceName)
	fmt.Fprintf(os.Stdout, "  env: %s\n", env)
	fmt.Fprintf(os.Stdout, "  listen: %s\n", addr)
	fmt.Fprintf(os.Stdout, "  grpc: %s:%d\n", host, port)
	fmt.Fprintf(os.Stdout, "  config: services.shop.port\n\n")
}

func shutdownOnSignal(server *grpc.Server, log *zap.Logger) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	log.Info("service shutting down", zap.String("service", serviceName))
	server.GracefulStop()
}
