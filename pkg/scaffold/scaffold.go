package scaffold

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// InitOptions controls how bw-cli generates a new project from this scaffold.
type InitOptions struct {
	SourceDir   string
	TargetDir   string
	ModulePath  string
	RepoURL     string
	Branch      string
	RunTidy     bool
	IncludeDemo bool
}

// Init copies or clones the scaffold, then rewrites module paths for the target project.
func Init(opts InitOptions) error {
	if opts.TargetDir == "" {
		return errors.New("target dir is required")
	}
	if opts.ModulePath == "" {
		return errors.New("module path is required")
	}

	if opts.RepoURL != "" {
		if err := clone(opts); err != nil {
			return err
		}
	} else {
		if opts.SourceDir == "" {
			return errors.New("source dir or repo url is required")
		}
		if err := copyDir(opts.SourceDir, opts.TargetDir); err != nil {
			return err
		}
	}

	oldModule, err := readModule(filepath.Join(opts.TargetDir, "go.mod"))
	if err != nil {
		return err
	}
	if err := rewriteModule(opts.TargetDir, oldModule, opts.ModulePath); err != nil {
		return err
	}
	if err := removeScaffoldTooling(opts.TargetDir); err != nil {
		return err
	}
	if opts.IncludeDemo {
		if err := writeDemoDocs(opts.TargetDir, opts.ModulePath); err != nil {
			return err
		}
	} else {
		if err := stripDemo(opts.TargetDir, opts.ModulePath); err != nil {
			return err
		}
	}
	if opts.RunTidy {
		cmd := exec.Command("go", "mod", "tidy")
		cmd.Dir = opts.TargetDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go mod tidy: %w", err)
		}
	}
	return nil
}

func clone(opts InitOptions) error {
	args := []string{"clone", "--depth", "1"}
	if opts.Branch != "" {
		args = append(args, "--branch", opts.Branch)
	}
	args = append(args, opts.RepoURL, opts.TargetDir)
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	// Generated projects should not inherit the scaffold repository remote.
	return os.RemoveAll(filepath.Join(opts.TargetDir, ".git"))
}

func copyDir(source string, target string) error {
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !sourceInfo.IsDir() {
		return fmt.Errorf("source %s is not a directory", source)
	}
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(target, 0o755)
		}
		if shouldSkip(rel, entry) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		dest := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		return copyFile(path, dest)
	})
}

func shouldSkip(rel string, entry os.DirEntry) bool {
	name := entry.Name()
	if name == ".git" || name == ".idea" || name == "data" || name == "logs" || name == "tmp" || name == ".DS_Store" {
		return true
	}
	if rel == filepath.Join("docs", "superpowers") {
		return true
	}
	if strings.HasSuffix(name, ".log") {
		return true
	}
	return false
}

func removeScaffoldTooling(root string) error {
	for _, rel := range []string{
		filepath.Join("cmd", "bw-cli"),
		filepath.Join("pkg", "scaffold"),
	} {
		if err := os.RemoveAll(filepath.Join(root, rel)); err != nil {
			return err
		}
	}
	return nil
}

func stripDemo(root string, module string) error {
	for _, item := range []struct {
		rel  string
		keep []string
	}{
		{rel: "cmd", keep: []string{"gateway"}},
		{rel: "internal", keep: []string{"gateway"}},
		{rel: filepath.Join("api", "proto")},
		{rel: filepath.Join("api", "gen")},
	} {
		if err := removeChildrenExcept(filepath.Join(root, item.rel), item.keep...); err != nil {
			return err
		}
	}
	for _, rel := range []string{
		filepath.Join("internal", "gateway", "client"),
		filepath.Join("internal", "gateway", "handler"),
		filepath.Join("internal", "gateway", "request"),
		filepath.Join("internal", "gateway", "router", "user_routes.go"),
		filepath.Join("internal", "gateway", "router", "note_routes.go"),
		filepath.Join("internal", "gateway", "router", "router_test.go"),
	} {
		if err := os.RemoveAll(filepath.Join(root, rel)); err != nil {
			return err
		}
	}
	if err := writeCleanGateway(root, module); err != nil {
		return err
	}
	if err := writeCleanMakefile(root); err != nil {
		return err
	}
	if err := writeCleanConfig(root); err != nil {
		return err
	}
	if err := writeCleanDocs(root, module); err != nil {
		return err
	}
	return nil
}

func removeChildrenExcept(dir string, keep ...string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	keepSet := make(map[string]struct{}, len(keep))
	for _, name := range keep {
		keepSet[name] = struct{}{}
	}
	for _, entry := range entries {
		if _, ok := keepSet[entry.Name()]; ok {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func writeCleanGateway(root string, module string) error {
	if exists(filepath.Join(root, "cmd", "gateway")) {
		if err := os.WriteFile(filepath.Join(root, "cmd", "gateway", "main.go"), []byte(cleanGatewayMain(module)), 0o644); err != nil {
			return err
		}
	}
	routerDir := filepath.Join(root, "internal", "gateway", "router")
	if exists(routerDir) {
		if err := os.WriteFile(filepath.Join(routerDir, "router.go"), []byte(cleanRouter(module)), 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(routerDir, "v1.go"), []byte(cleanV1Router()), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func writeCleanMakefile(root string) error {
	path := filepath.Join(root, "Makefile")
	if !exists(path) {
		return nil
	}
	return os.WriteFile(path, []byte(cleanMakefile()), 0o644)
}

func writeCleanConfig(root string) error {
	path := filepath.Join(root, "configs", "config.yaml")
	if !exists(path) {
		return nil
	}
	if err := os.WriteFile(path, []byte(cleanConfigYAML()), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "configs", "nacos.yaml"), []byte(cleanNacosYAML()), 0o644)
}

func writeCleanDocs(root string, module string) error {
	if exists(filepath.Join(root, "README.md")) {
		if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(cleanREADME(module)), 0o644); err != nil {
			return err
		}
	}
	docsDir := filepath.Join(root, "docs")
	if !exists(docsDir) {
		return nil
	}
	if err := os.WriteFile(filepath.Join(docsDir, "usage.md"), []byte(cleanUsageDoc(module)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(docsDir, "architecture.md"), []byte(cleanArchitectureDoc()), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(docsDir, "toolkit.md"), []byte(generatedToolkitDoc(module)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(docsDir, "mongodb.md"), []byte(generatedMongoDBDoc(module)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(docsDir, "alipay.md"), []byte(generatedAlipayDoc()), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(docsDir, "elasticsearch.md"), []byte(generatedElasticsearchDoc()), 0o644)
}

func writeDemoDocs(root string, module string) error {
	if exists(filepath.Join(root, "README.md")) {
		if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(demoREADME(module)), 0o644); err != nil {
			return err
		}
	}
	docsDir := filepath.Join(root, "docs")
	if !exists(docsDir) {
		return nil
	}
	if err := os.WriteFile(filepath.Join(docsDir, "usage.md"), []byte(demoUsageDoc(module)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(docsDir, "architecture.md"), []byte(demoArchitectureDoc()), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(docsDir, "toolkit.md"), []byte(generatedToolkitDoc(module)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(docsDir, "mongodb.md"), []byte(generatedMongoDBDoc(module)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(docsDir, "alipay.md"), []byte(generatedAlipayDoc()), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(docsDir, "elasticsearch.md"), []byte(generatedElasticsearchDoc()), 0o644)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func cleanGatewayMain(module string) string {
	return fmt.Sprintf(`package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"%s/internal/gateway/router"
	"%s/pkg/config"
	"%s/pkg/logger"
	"%s/pkg/observability"
)

func main() {
	// Load runtime settings before constructing dependencies.
	if err := config.InitGlobal("configs/config.yaml"); err != nil {
		panic(err)
	}
	cfg := config.MustGlobal()
	gatewayServiceName := cfg.ServiceName("gateway")
	cfg.Log.Service = gatewayServiceName
	cfg.Log = logger.WithDailyFileName(cfg.Log, time.Now())

	log, err := logger.New(cfg.Log)
	if err != nil {
		panic(err)
	}
	defer log.Sync()
	observability.Register(gatewayServiceName, log)
	config.PrintSourceNotice(cfg, os.Stdout)

	engine := router.New(log, cfg.Middleware)
	addr := fmt.Sprintf("%%s:%%d", cfg.HTTP.Host, cfg.HTTP.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      engine,
		ReadTimeout:  time.Duration(cfg.HTTP.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.HTTP.WriteTimeoutSeconds) * time.Second,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		printStartupFailure(addr, err)
		log.Fatal("gateway listen failed", zap.String("addr", addr), zap.Error(err))
	}
	printStartupSummary(cfg, addr)

	go func() {
		log.Info("gateway listening", zap.String("addr", addr))
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatal("gateway stopped unexpectedly", zap.Error(err))
		}
	}()

	waitForShutdown(server, log)
}

func printStartupFailure(addr string, err error) {
	fmt.Fprintf(os.Stderr, "\n[Gateway Start Failed]\n")
	fmt.Fprintf(os.Stderr, "  listen: %%s\n", addr)
	fmt.Fprintf(os.Stderr, "  error: %%v\n\n", err)
}

func printStartupSummary(cfg *config.Config, addr string) {
	host := cfg.HTTP.Host
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	baseURL := fmt.Sprintf("http://%%s:%%d", host, cfg.HTTP.Port)
	fmt.Fprintf(os.Stdout, "\n[Gateway Started]\n")
	fmt.Fprintf(os.Stdout, "  service: %%s\n", cfg.ServiceName("gateway"))
	fmt.Fprintf(os.Stdout, "  env: %%s\n", cfg.App.Env)
	fmt.Fprintf(os.Stdout, "  listen: %%s\n", addr)
	fmt.Fprintf(os.Stdout, "  http: %%s\n", baseURL)
	fmt.Fprintf(os.Stdout, "  health: %%s/healthz\n", baseURL)
	fmt.Fprintf(os.Stdout, "  api: %%s/api/v1\n\n", baseURL)
}

func waitForShutdown(server *http.Server, log *zap.Logger) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	log.Info("gateway shutting down")
	if err := server.Shutdown(ctx); err != nil {
		log.Error("gateway shutdown failed", zap.Error(err))
	}
}
`, module, module, module, module)
}

func cleanRouter(module string) string {
	return fmt.Sprintf(`// Package router owns Gin engine construction and route registration.
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"%s/pkg/config"
	"%s/pkg/middleware"
)

// New builds the gateway Gin engine with configured middleware and versioned API routes.
func New(log *zap.Logger, middlewareCfg config.MiddlewareConfig) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(
		middleware.CORS(middlewareCfg.CORS),
		middleware.RequestID(),
		middleware.RequestLogger(log),
		gin.Recovery(),
	)
	r.OPTIONS("/*path", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	registerHealthRoutes(r)
	registerAPIRoutes(r)
	return r
}
`, module, module)
}

func cleanV1Router() string {
	return `package router

import "github.com/gin-gonic/gin"

// registerAPIRoutes creates the /api/v1 route namespace.
// Add business-specific route files beside this file as services are introduced.
func registerAPIRoutes(r *gin.Engine) {
	api := r.Group("/api")
	v1 := api.Group("/v1")
	_ = v1
}
`
}

func cleanMakefile() string {
	return `GO ?= go
PROTOC ?= protoc
PROTO_PATH ?= api/proto
PROTO_OUT ?= api/gen

export PROTOC
export PROTO_PATH
export PROTO_OUT

.PHONY: proto test tidy run-gateway tools

tools:
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

proto:
	$(GO) run ./tools/protogen

test:
	$(GO) test ./...

tidy:
	$(GO) mod tidy

run-gateway:
	$(GO) run ./cmd/gateway
`
}

func cleanConfigYAML() string {
	return `app:
  name: app
  env: local

http:
  host: 0.0.0.0
  port: 8080
  read_timeout_seconds: 5
  write_timeout_seconds: 10

grpc:
  host: 0.0.0.0

services:
  gateway:
    name: gateway

database:
  driver: sqlite
  dsn: data/app.db

mysql:
  dsn: ""
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime_seconds: 3600

postgresql:
  dsn: ""
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime_seconds: 3600

mongodb:
  uri: mongodb://127.0.0.1:27017
  username: ""
  password: ""
  database: app
  app_name: bw-cli
  min_pool_size: 0
  max_pool_size: 100
  connect_timeout_seconds: 10
  server_selection_timeout_seconds: 5

file_storage:
  provider: minio
  max_size_mb: 100
  object_prefix: uploads
  public_base_url: ""
  allowed_extensions:
    - .doc
    - .docx
    - .pdf
    - .jpg
    - .jpeg
    - .png
    - .gif
    - .webp
    - .bmp
    - .svg
    - .mp4
    - .mov
    - .avi
    - .mkv
    - .webm
    - .mp3
    - .wav
    - .ogg
    - .m4a
    - .flac
    - .aac
  allowed_content_types:
    - application/msword
    - application/vnd.openxmlformats-officedocument.wordprocessingml.document
    - application/pdf
    - image/jpeg
    - image/png
    - image/gif
    - image/webp
    - image/bmp
    - image/svg+xml
    - video/mp4
    - video/quicktime
    - video/x-msvideo
    - video/x-matroska
    - video/webm
    - audio/mpeg
    - audio/wav
    - audio/x-wav
    - audio/ogg
    - audio/mp4
    - audio/flac
    - audio/aac
  minio:
    endpoint: ""
    access_key_id: ""
    secret_access_key: ""
    bucket: ""
    region: ""
    use_ssl: false
  oss:
    endpoint: ""
    access_key_id: ""
    access_key_secret: ""
    bucket: ""
  qiniu:
    access_key: ""
    secret_key: ""
    bucket: ""
    region: ""
    use_https: true
    use_cdn_domains: false
  cos:
    secret_id: ""
    secret_key: ""
    bucket: ""
    region: ""
    bucket_url: ""

redis:
  addr: 127.0.0.1:6379
  username: ""
  password: ""
  db: 0
  pool_size: 10
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s
  lock:
    key_prefix: app
    default_ttl: 30s

elasticsearch:
  addresses:
    - http://127.0.0.1:9200
  username: ""
  password: ""
  cloud_id: ""
  api_key: ""

kafka:
  brokers:
    - 127.0.0.1:9092
  topic: app-events
  group_id: app-consumer
  client_id: app
  required_acks: all
  dial_timeout: 5s
  producer:
    max_attempts: 10
    batch_size: 100
    batch_bytes: 1048576
    batch_timeout: 10ms
    read_timeout: 10s
    write_timeout: 10s
    async: false
    compression: none
    allow_auto_topic_creation: false
  consumer:
    queue_capacity: 100
    min_bytes: 1
    max_bytes: 10485760
    max_wait: 10s
    read_batch_timeout: 10s
    commit_interval: 0s
    heartbeat_interval: 3s
    session_timeout: 30s
    rebalance_timeout: 30s
    start_offset: first
    watch_partition_changes: true
    max_attempts: 3
  sasl:
    enable: false
    mechanism: plain
    username: ""
    password: ""
  tls:
    enable: false
    insecure_skip_verify: false
    server_name: ""

alipay:
  app_id: ""
  private_key: ""
  alipay_public_key: ""
  production: false
  notify_url: ""
  return_url: ""
  encrypt_key: ""
  app_cert_public_key_path: ""
  alipay_root_cert_path: ""
  alipay_cert_public_key_path: ""

middleware:
  cors:
    allow_origins:
      - "*"
    allow_methods:
      - GET
      - POST
      - PUT
      - PATCH
      - DELETE
      - OPTIONS
    allow_headers:
      - Origin
      - Content-Type
      - Authorization
      - X-Request-ID
    allow_credentials: false
  jwt:
    secret: ""
    issuer: app
    expire_seconds: 7200

log:
  service: app
  environment: local
  level: info
  encoding: json
  file:
    enabled: true
    filename: logs/app.log
    max_size_mb: 128
    max_backups: 14
    max_age_days: 7
    compress: true
`
}

func cleanNacosYAML() string {
	return `enabled: false
server_addr: 127.0.0.1
server_port: 8848
namespace_id: ""
group: DEFAULT_GROUP
data_id: xiaolanshu.yaml
username: ""
password: ""
timeout_ms: 5000
log_dir: logs/nacos
cache_dir: data/nacos/cache
log_level: info
fail_fast: false
watch: false
`
}

func copyFile(source string, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func readModule(goModPath string) (string, error) {
	file, err := os.Open(goModPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("module directive not found")
}

func rewriteModule(root string, oldModule string, newModule string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !shouldRewrite(path) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		updated := strings.ReplaceAll(string(data), oldModule, newModule)
		return os.WriteFile(path, []byte(updated), 0o644)
	})
}

func shouldRewrite(path string) bool {
	if strings.HasSuffix(path, ".pb.go") {
		return false
	}
	switch filepath.Ext(path) {
	case ".go", ".mod", ".md", ".yaml", ".yml", ".proto":
		return true
	default:
		return false
	}
}
