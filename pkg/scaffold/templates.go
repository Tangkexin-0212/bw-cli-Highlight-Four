package scaffold

const serviceProtoTemplate = `syntax = "proto3";

package {{ .ProtoPackage }};

option go_package = "{{ .Module }}/api/gen/{{ .Dir }}/v1;{{ .GoPackage }}";

// {{ .Pascal }}Service is the gRPC boundary for the {{ .Dir }} business service.
// The default CRUD contract is ready to call. Extend messages and RPCs as business grows.
service {{ .Pascal }}Service {
  rpc Create{{ .Pascal }}(Create{{ .Pascal }}Request) returns ({{ .Pascal }}Response);
  rpc Get{{ .Pascal }}(Get{{ .Pascal }}Request) returns ({{ .Pascal }}Response);
  rpc List{{ .Pascal }}s(List{{ .Pascal }}sRequest) returns (List{{ .Pascal }}sResponse);
  rpc Update{{ .Pascal }}(Update{{ .Pascal }}Request) returns ({{ .Pascal }}Response);
  rpc Delete{{ .Pascal }}(Delete{{ .Pascal }}Request) returns (Delete{{ .Pascal }}Response);
}

message Create{{ .Pascal }}Request {
  string name = 1;
  string description = 2;
}

message Get{{ .Pascal }}Request {
  string id = 1;
}

message List{{ .Pascal }}sRequest {
  int32 page = 1;
  int32 page_size = 2;
}

message Update{{ .Pascal }}Request {
  string id = 1;
  string name = 2;
  string description = 3;
}

message Delete{{ .Pascal }}Request {
  string id = 1;
}

message {{ .Pascal }}Response {
  string id = 1;
  string name = 2;
  string description = 3;
  string created_at = 4;
  string updated_at = 5;
}

message List{{ .Pascal }}sResponse {
  repeated {{ .Pascal }}Response items = 1;
  int64 total = 2;
}

message Delete{{ .Pascal }}Response {
  bool success = 1;
}
`

const serviceMainTemplate = `package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	{{ .GoPackage }} "{{ .Module }}/api/gen/{{ .Dir }}/v1"
	{{ .GoIdent }}handler "{{ .Module }}/internal/{{ .Dir }}/handler"
	{{ .GoIdent }}repo "{{ .Module }}/internal/{{ .Dir }}/repo"
	{{ .GoIdent }}service "{{ .Module }}/internal/{{ .Dir }}/service"
	"{{ .Module }}/pkg/config"
	"{{ .Module }}/pkg/database"
	"{{ .Module }}/pkg/grpcx"
	"{{ .Module }}/pkg/logger"
)

const serviceName = "{{ .ServiceName }}"
const defaultGRPCPort = {{ .Port }}

func main() {
	if err := config.InitGlobal("configs/config.yaml"); err != nil {
		panic(err)
	}
	cfg := config.MustGlobal()
	cfg.Log.Service = cfg.ServiceName("{{ .Dir }}")
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
	if err := {{ .GoIdent }}repo.AutoMigrate(db); err != nil {
		log.Fatal("migrate {{ .Dir }} database failed", zap.Error(err))
	}

	repo := {{ .GoIdent }}repo.NewGormRepository(db, log)
	svc := {{ .GoIdent }}service.NewService(repo, log)
	server := grpc.NewServer(grpc.UnaryInterceptor(grpcx.UnaryServerInterceptor(log)))
	{{ .GoPackage }}.Register{{ .Pascal }}ServiceServer(server, {{ .GoIdent }}handler.NewServer(svc, log))

	port := cfg.ServicePort("{{ .Dir }}", defaultGRPCPort)
	addr := fmt.Sprintf("%s:%d", cfg.GRPC.Host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n[Service Start Failed]\n  service: %s\n  listen: %s\n  error: %v\n\n", serviceName, addr, err)
		log.Fatal("listen failed", zap.String("addr", addr), zap.Error(err))
	}

	printStartupSummary(cfg.App.Env, addr, port)
	go shutdownOnSignal(server, log)
	if err := server.Serve(listener); err != nil {
		log.Fatal("service stopped unexpectedly", zap.Error(err))
	}
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
	fmt.Fprintf(os.Stdout, "  config: services.{{ .Dir }}.port\n\n")
}

func shutdownOnSignal(server *grpc.Server, log *zap.Logger) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	log.Info("service shutting down", zap.String("service", serviceName))
	server.GracefulStop()
}
`

const serviceModelTemplate = `package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	Err{{ .Pascal }}NotFound = errors.New("{{ .Dir }} not found")
	ErrInvalid{{ .Pascal }} = errors.New("invalid {{ .Dir }}")
)

// {{ .Pascal }} is the aggregate root for the {{ .Dir }} business service.
// Replace Name and Description with real business fields when the domain is clear.
type {{ .Pascal }} struct {
	ID        string
	Name        string
	Description string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// New{{ .Pascal }} validates input and creates an aggregate with framework-managed identity fields.
func New{{ .Pascal }}(name string, description string) (*{{ .Pascal }}, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" {
		return nil, ErrInvalid{{ .Pascal }}
	}
	now := time.Now().UTC()
	return &{{ .Pascal }}{
		ID:          uuid.NewString(),
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Update changes mutable fields while keeping validation inside the domain model.
func (item *{{ .Pascal }}) Update(name string, description string) error {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if item == nil || item.ID == "" || name == "" {
		return ErrInvalid{{ .Pascal }}
	}
	item.Name = name
	item.Description = description
	item.UpdatedAt = time.Now().UTC()
	return nil
}
`

const serviceRepositoryTemplate = `package model

import "context"

// Repository defines persistence behavior required by the {{ .Dir }} service layer.
type Repository interface {
	Save(ctx context.Context, item *{{ .Pascal }}) error
	FindByID(ctx context.Context, id string) (*{{ .Pascal }}, error)
	List(ctx context.Context, offset int, limit int) ([]*{{ .Pascal }}, int64, error)
	Delete(ctx context.Context, id string) error
}
`

const serviceCommandTemplate = `package dto

// CreateCommand contains input for creating a {{ .Dir }} record.
type CreateCommand struct {
	Name        string
	Description string
}

// UpdateCommand contains input for updating a {{ .Dir }} record.
type UpdateCommand struct {
	ID          string
	Name        string
	Description string
}

// ListCommand contains pagination input for listing {{ .Dir }} records.
type ListCommand struct {
	Page     int32
	PageSize int32
}
`

const serviceDTOTemplate = `package dto

import (
	"time"

	"{{ .Module }}/internal/{{ .Dir }}/model"
)

// {{ .Pascal }}DTO is returned by use cases and converted by handlers.
type {{ .Pascal }}DTO struct {
	ID          string
	Name        string
	Description string
	CreatedAt   string
	UpdatedAt   string
}

// List{{ .Pascal }}DTO contains paginated list output.
type List{{ .Pascal }}DTO struct {
	Items []*{{ .Pascal }}DTO
	Total int64
}

// From{{ .Pascal }} converts a {{ .Dir }} aggregate into the service response DTO.
func From{{ .Pascal }}(item *model.{{ .Pascal }}) *{{ .Pascal }}DTO {
	return &{{ .Pascal }}DTO{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		CreatedAt:   formatTime(item.CreatedAt),
		UpdatedAt:   formatTime(item.UpdatedAt),
	}
}

// formatTime keeps zero time empty and serializes real values in a stable API format.
func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}
`

const serviceUseCaseTemplate = `package service

import (
	"context"

	"go.uber.org/zap"

	"{{ .Module }}/internal/{{ .Dir }}/dto"
	"{{ .Module }}/internal/{{ .Dir }}/model"
)

// Service orchestrates {{ .Dir }} use cases.
type Service struct {
	repo model.Repository
	log  *zap.Logger
}

// NewService constructs the {{ .Dir }} use-case service.
func NewService(repo model.Repository, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{repo: repo, log: log}
}

// Create creates a {{ .Dir }} record.
func (s *Service) Create(ctx context.Context, cmd dto.CreateCommand) (*dto.{{ .Pascal }}DTO, error) {
	item, err := model.New{{ .Pascal }}(cmd.Name, cmd.Description)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, item); err != nil {
		return nil, err
	}
	s.log.Info("{{ .Dir }} created", zap.String("aggregate_id", item.ID), zap.String("use_case", "Create{{ .Pascal }}"))
	return dto.From{{ .Pascal }}(item), nil
}

// Get returns one {{ .Dir }} record by id.
func (s *Service) Get(ctx context.Context, id string) (*dto.{{ .Pascal }}DTO, error) {
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.From{{ .Pascal }}(item), nil
}

// List returns paginated {{ .Dir }} records.
func (s *Service) List(ctx context.Context, cmd dto.ListCommand) (*dto.List{{ .Pascal }}DTO, error) {
	offset, limit := normalizePagination(cmd.Page, cmd.PageSize)
	items, total, err := s.repo.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	output := &dto.List{{ .Pascal }}DTO{Items: make([]*dto.{{ .Pascal }}DTO, 0, len(items)), Total: total}
	for _, item := range items {
		output.Items = append(output.Items, dto.From{{ .Pascal }}(item))
	}
	return output, nil
}

// Update changes one {{ .Dir }} record by id.
func (s *Service) Update(ctx context.Context, cmd dto.UpdateCommand) (*dto.{{ .Pascal }}DTO, error) {
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
	s.log.Info("{{ .Dir }} updated", zap.String("aggregate_id", item.ID), zap.String("use_case", "Update{{ .Pascal }}"))
	return dto.From{{ .Pascal }}(item), nil
}

// Delete removes one {{ .Dir }} record by id.
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.log.Info("{{ .Dir }} deleted", zap.String("aggregate_id", id), zap.String("use_case", "Delete{{ .Pascal }}"))
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
`
