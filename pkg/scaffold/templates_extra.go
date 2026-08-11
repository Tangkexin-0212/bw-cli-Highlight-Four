package scaffold

const serviceUseCaseTestTemplate = `package service

import (
	"context"
	"testing"

	"{{ .Module }}/internal/{{ .Dir }}/dto"
	"{{ .Module }}/internal/{{ .Dir }}/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewService(t *testing.T) {
	svc := NewService(nil, zap.NewNop())

	require.NotNil(t, svc)
}

func TestServiceCRUD(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepository()
	svc := NewService(repo, zap.NewNop())

	created, err := svc.Create(ctx, dto.CreateCommand{Name: "first", Description: "created from service test"})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.Equal(t, "first", created.Name)

	got, err := svc.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)

	list, err := svc.List(ctx, dto.ListCommand{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), list.Total)
	require.Len(t, list.Items, 1)

	updated, err := svc.Update(ctx, dto.UpdateCommand{ID: created.ID, Name: "updated", Description: "updated from service test"})
	require.NoError(t, err)
	require.Equal(t, "updated", updated.Name)

	require.NoError(t, svc.Delete(ctx, created.ID))
	_, err = svc.Get(ctx, created.ID)
	require.ErrorIs(t, err, model.Err{{ .Pascal }}NotFound)
}

type fakeRepository struct {
	items map[string]*model.{{ .Pascal }}
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{items: make(map[string]*model.{{ .Pascal }})}
}

func (r *fakeRepository) Save(ctx context.Context, item *model.{{ .Pascal }}) error {
	copy := *item
	r.items[item.ID] = &copy
	return nil
}

func (r *fakeRepository) FindByID(ctx context.Context, id string) (*model.{{ .Pascal }}, error) {
	item, ok := r.items[id]
	if !ok {
		return nil, model.Err{{ .Pascal }}NotFound
	}
	copy := *item
	return &copy, nil
}

func (r *fakeRepository) List(ctx context.Context, offset int, limit int) ([]*model.{{ .Pascal }}, int64, error) {
	items := make([]*model.{{ .Pascal }}, 0, len(r.items))
	for _, item := range r.items {
		copy := *item
		items = append(items, &copy)
	}
	if offset > len(items) {
		return nil, int64(len(items)), nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end], int64(len(items)), nil
}

func (r *fakeRepository) Delete(ctx context.Context, id string) error {
	if _, ok := r.items[id]; !ok {
		return model.Err{{ .Pascal }}NotFound
	}
	delete(r.items, id)
	return nil
}

var _ model.Repository = (*fakeRepository)(nil)
`

const serviceHandlerTemplate = `package handler

import (
	"context"
	stderrors "errors"

	"go.uber.org/zap"

	{{ .GoPackage }} "{{ .Module }}/api/gen/{{ .Dir }}/v1"
	"{{ .Module }}/internal/{{ .Dir }}/dto"
	"{{ .Module }}/internal/{{ .Dir }}/model"
	"{{ .Module }}/internal/{{ .Dir }}/service"
	apperrors "{{ .Module }}/pkg/errors"
)

// Server adapts {{ .Dir }} gRPC requests to service use cases.
type Server struct {
	{{ .GoPackage }}.Unimplemented{{ .Pascal }}ServiceServer
	svc *service.Service
	log *zap.Logger
}

// NewServer constructs the {{ .Dir }} gRPC server adapter.
func NewServer(svc *service.Service, log *zap.Logger) *Server {
	if log == nil {
		log = zap.NewNop()
	}
	return &Server{svc: svc, log: log}
}

// Create{{ .Pascal }} handles the create RPC.
func (s *Server) Create{{ .Pascal }}(ctx context.Context, req *{{ .GoPackage }}.Create{{ .Pascal }}Request) (*{{ .GoPackage }}.{{ .Pascal }}Response, error) {
	item, err := s.svc.Create(ctx, dto.CreateCommand{
		Name:        req.GetName(),
		Description: req.GetDescription(),
	})
	if err != nil {
		return nil, map{{ .Pascal }}Error(err)
	}
	return toProto(item), nil
}

// Get{{ .Pascal }} handles lookup by id.
func (s *Server) Get{{ .Pascal }}(ctx context.Context, req *{{ .GoPackage }}.Get{{ .Pascal }}Request) (*{{ .GoPackage }}.{{ .Pascal }}Response, error) {
	item, err := s.svc.Get(ctx, req.GetId())
	if err != nil {
		return nil, map{{ .Pascal }}Error(err)
	}
	return toProto(item), nil
}

// List{{ .Pascal }}s handles paginated listing.
func (s *Server) List{{ .Pascal }}s(ctx context.Context, req *{{ .GoPackage }}.List{{ .Pascal }}sRequest) (*{{ .GoPackage }}.List{{ .Pascal }}sResponse, error) {
	list, err := s.svc.List(ctx, dto.ListCommand{
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	})
	if err != nil {
		return nil, map{{ .Pascal }}Error(err)
	}
	resp := &{{ .GoPackage }}.List{{ .Pascal }}sResponse{
		Items: make([]*{{ .GoPackage }}.{{ .Pascal }}Response, 0, len(list.Items)),
		Total: list.Total,
	}
	for _, item := range list.Items {
		resp.Items = append(resp.Items, toProto(item))
	}
	return resp, nil
}

// Update{{ .Pascal }} handles updates by id.
func (s *Server) Update{{ .Pascal }}(ctx context.Context, req *{{ .GoPackage }}.Update{{ .Pascal }}Request) (*{{ .GoPackage }}.{{ .Pascal }}Response, error) {
	item, err := s.svc.Update(ctx, dto.UpdateCommand{
		ID:          req.GetId(),
		Name:        req.GetName(),
		Description: req.GetDescription(),
	})
	if err != nil {
		return nil, map{{ .Pascal }}Error(err)
	}
	return toProto(item), nil
}

// Delete{{ .Pascal }} handles deletion by id.
func (s *Server) Delete{{ .Pascal }}(ctx context.Context, req *{{ .GoPackage }}.Delete{{ .Pascal }}Request) (*{{ .GoPackage }}.Delete{{ .Pascal }}Response, error) {
	if err := s.svc.Delete(ctx, req.GetId()); err != nil {
		return nil, map{{ .Pascal }}Error(err)
	}
	return &{{ .GoPackage }}.Delete{{ .Pascal }}Response{Success: true}, nil
}

func toProto(item *dto.{{ .Pascal }}DTO) *{{ .GoPackage }}.{{ .Pascal }}Response {
	return &{{ .GoPackage }}.{{ .Pascal }}Response{
		Id:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func map{{ .Pascal }}Error(err error) error {
	switch {
	case stderrors.Is(err, model.ErrInvalid{{ .Pascal }}):
		return apperrors.InvalidArgument("invalid_{{ .Dir }}", "invalid {{ .Dir }} input")
	case stderrors.Is(err, model.Err{{ .Pascal }}NotFound):
		return apperrors.NotFound("{{ .Dir }}_not_found", "{{ .Dir }} not found")
	default:
		return apperrors.Wrap(apperrors.KindInternal, "{{ .Dir }}_service_error", "{{ .Dir }} service error", err)
	}
}
`

const gatewayClientsTemplate = `package client

import (
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	{{ .GoPackage }} "{{ .Module }}/api/gen/{{ .Dir }}/v1"
	"{{ .Module }}/pkg/config"
)

// Clients groups all gRPC clients used by the HTTP gateway.
type Clients struct {
	{{ .Pascal }}  {{ .GoPackage }}.{{ .Pascal }}ServiceClient
	Config *config.Config

	conns []*grpc.ClientConn
}

// New dials configured gRPC targets and builds typed service clients.
func New(cfg *config.Config, log *zap.Logger) (*Clients, error) {
	{{ .GoIdent }}Target := cfg.ServiceTarget("{{ .Dir }}")
	{{ .GoIdent }}Conn, err := grpc.Dial({{ .GoIdent }}Target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial {{ .Dir }} service: %w", err)
	}

	log.Info("grpc clients initialized",
		zap.String("{{ .Dir }}_target", {{ .GoIdent }}Target),
	)
	return &Clients{
		{{ .Pascal }}:  {{ .GoPackage }}.New{{ .Pascal }}ServiceClient({{ .GoIdent }}Conn),
		Config: cfg,
		conns:  []*grpc.ClientConn{ {{ .GoIdent }}Conn },
	}, nil
}

// Close releases all gateway gRPC client connections.
func (c *Clients) Close() {
	for _, conn := range c.conns {
		_ = conn.Close()
	}
}
`

const gatewayCommonTemplate = `package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"

	"{{ .Module }}/pkg/grpcx"
	"{{ .Module }}/pkg/httpx"
)

// outgoingContext forwards gateway metadata such as request id to downstream gRPC calls.
func outgoingContext(c *gin.Context) context.Context {
	return metadata.AppendToOutgoingContext(c.Request.Context(), grpcx.MetadataRequestID, httpx.RequestID(c))
}
`

const gatewayRoutesTemplate = `package router

import (
	"github.com/gin-gonic/gin"

	"{{ .Module }}/internal/gateway/handler"
)

// register{{ .Pascal }}Routes registers /api/v1/{{ .TableName }} endpoints in one business-specific file.
func register{{ .Pascal }}Routes(v1 *gin.RouterGroup, {{ .GoIdent }}Handler *handler.{{ .Pascal }}Handler) {
	routes := v1.Group("/{{ .TableName }}")
	routes.POST("", {{ .GoIdent }}Handler.Create)
	routes.GET("", {{ .GoIdent }}Handler.List)
	routes.GET("/:id", {{ .GoIdent }}Handler.Get)
	routes.PUT("/:id", {{ .GoIdent }}Handler.Update)
	routes.DELETE("/:id", {{ .GoIdent }}Handler.Delete)
}
`

const cleanGatewayV1WithServiceTemplate = `package router

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"{{ .Module }}/internal/gateway/client"
	"{{ .Module }}/internal/gateway/handler"
)

// registerAPIRoutes creates the /api/v1 route namespace before delegating by business module.
func registerAPIRoutes(r *gin.Engine, clients *client.Clients, log *zap.Logger) {
	api := r.Group("/api")
	v1 := api.Group("/v1")

	register{{ .Pascal }}Routes(v1, handler.New{{ .Pascal }}Handler(clients.{{ .Pascal }}, log))
}
`

const gatewayHandlerTemplate = `package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	{{ .GoPackage }} "{{ .Module }}/api/gen/{{ .Dir }}/v1"
	"{{ .Module }}/internal/gateway/request"
	apperrors "{{ .Module }}/pkg/errors"
	"{{ .Module }}/pkg/httpx"
)

// {{ .Pascal }}Handler adapts {{ .Dir }} HTTP endpoints to the generated gRPC client.
type {{ .Pascal }}Handler struct {
	client {{ .GoPackage }}.{{ .Pascal }}ServiceClient
	log    *zap.Logger
}

// New{{ .Pascal }}Handler wires the {{ .Dir }} gRPC client into HTTP handler methods.
func New{{ .Pascal }}Handler(client {{ .GoPackage }}.{{ .Pascal }}ServiceClient, log *zap.Logger) *{{ .Pascal }}Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return &{{ .Pascal }}Handler{
		client: client,
		log:    log,
	}
}

// Create proxies POST /api/v1/{{ .TableName }} to Create{{ .Pascal }}.
func (h *{{ .Pascal }}Handler) Create(c *gin.Context) {
	var req request.Create{{ .Pascal }}Request
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.Create{{ .Pascal }}(outgoingContext(c), &{{ .GoPackage }}.Create{{ .Pascal }}Request{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	h.log.Info("gateway {{ .Dir }} create proxied", zap.String("request_id", httpx.RequestID(c)), zap.String("aggregate_id", resp.GetId()))
	httpx.Created(c, resp)
}

// Get proxies GET /api/v1/{{ .TableName }}/:id to Get{{ .Pascal }}.
func (h *{{ .Pascal }}Handler) Get(c *gin.Context) {
	resp, err := h.client.Get{{ .Pascal }}(outgoingContext(c), &{{ .GoPackage }}.Get{{ .Pascal }}Request{Id: c.Param("id")})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// List proxies GET /api/v1/{{ .TableName }} to List{{ .Pascal }}s.
func (h *{{ .Pascal }}Handler) List(c *gin.Context) {
	var req request.List{{ .Pascal }}Request
	if err := c.ShouldBindQuery(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.List{{ .Pascal }}s(outgoingContext(c), &{{ .GoPackage }}.List{{ .Pascal }}sRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	httpx.OK(c, resp)
}

// Update proxies PUT /api/v1/{{ .TableName }}/:id to Update{{ .Pascal }}.
func (h *{{ .Pascal }}Handler) Update(c *gin.Context) {
	var req request.Update{{ .Pascal }}Request
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, apperrors.InvalidArgument("invalid_request", err.Error()))
		return
	}
	resp, err := h.client.Update{{ .Pascal }}(outgoingContext(c), &{{ .GoPackage }}.Update{{ .Pascal }}Request{
		Id:          c.Param("id"),
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	h.log.Info("gateway {{ .Dir }} update proxied", zap.String("request_id", httpx.RequestID(c)), zap.String("aggregate_id", resp.GetId()))
	httpx.OK(c, resp)
}

// Delete proxies DELETE /api/v1/{{ .TableName }}/:id to Delete{{ .Pascal }}.
func (h *{{ .Pascal }}Handler) Delete(c *gin.Context) {
	resp, err := h.client.Delete{{ .Pascal }}(outgoingContext(c), &{{ .GoPackage }}.Delete{{ .Pascal }}Request{Id: c.Param("id")})
	if err != nil {
		httpx.Error(c, apperrors.FromGRPC(err))
		return
	}
	h.log.Info("gateway {{ .Dir }} delete proxied", zap.String("request_id", httpx.RequestID(c)), zap.String("aggregate_id", c.Param("id")))
	httpx.OK(c, resp)
}
`
