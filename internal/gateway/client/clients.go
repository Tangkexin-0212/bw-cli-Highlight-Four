package client

import (
	commentv1 "github.com/BwCloudWeGo/bw-cli/api/gen/comment/v1"
	"fmt"

	shopv1 "github.com/BwCloudWeGo/bw-cli/api/gen/shop/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	notev1 "github.com/BwCloudWeGo/bw-cli/api/gen/note/v1"
	orderv1 "github.com/BwCloudWeGo/bw-cli/api/gen/order/v1"
	userv1 "github.com/BwCloudWeGo/bw-cli/api/gen/user/v1"
	"github.com/BwCloudWeGo/bw-cli/pkg/config"
)

// Clients groups all gRPC clients used by the HTTP gateway.
type Clients struct {
	Comment  commentv1.CommentServiceClient
	User   userv1.UserServiceClient
	Note   notev1.NoteServiceClient
	Order  orderv1.OrderServiceClient
	Shop   shopv1.ShopServiceClient
	Config *config.Config

	conns []*grpc.ClientConn
}

// New dials configured gRPC targets and builds typed service clients.
func New(cfg *config.Config, log *zap.Logger) (*Clients, error) {
	userTarget := cfg.ServiceTarget("user")
	noteTarget := cfg.ServiceTarget("note")
	orderTarget := cfg.ServiceTarget("order")
	shopTarget := cfg.ServiceTarget("shop")
	commentTarget := cfg.ServiceTarget("comment")
	userConn, err := grpc.Dial(userTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial user service: %w", err)
	}
	noteConn, err := grpc.Dial(noteTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		userConn.Close()
		return nil, fmt.Errorf("dial note service: %w", err)
	}
	orderConn, err := grpc.Dial(orderTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		userConn.Close()
		noteConn.Close()
		return nil, fmt.Errorf("dial order service: %w", err)
	}

	shopConn, err := grpc.Dial(shopTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		userConn.Close()
		noteConn.Close()
		orderConn.Close()
		return nil, fmt.Errorf("dial shop service: %w", err)
	}

	commentConn, err := grpc.Dial(commentTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		userConn.Close()
		noteConn.Close()
		orderConn.Close()
		shopConn.Close()
		return nil, fmt.Errorf("dial comment service: %w", err)
	}

	log.Info("grpc clients initialized",
		zap.String("user_target", userTarget),
		zap.String("note_target", noteTarget),
		zap.String("order_target", orderTarget),
		zap.String("comment_target", commentTarget),
	)
	return &Clients{
		User:   userv1.NewUserServiceClient(userConn),
		Note:   notev1.NewNoteServiceClient(noteConn),
		Order:  orderv1.NewOrderServiceClient(orderConn),
		Shop:   shopv1.NewShopServiceClient(shopConn),
		Comment:  commentv1.NewCommentServiceClient(commentConn),
		Config: cfg,
		conns:  []*grpc.ClientConn{userConn, noteConn, orderConn, shopConn, commentConn},
	}, nil
}

// Close releases all gateway gRPC client connections.
func (c *Clients) Close() {
	for _, conn := range c.conns {
		_ = conn.Close()
	}
}
