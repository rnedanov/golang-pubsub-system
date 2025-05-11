package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/rnedanov/golang-pubsub-system/gen"
	"github.com/rnedanov/golang-pubsub-system/internal/config"
	"github.com/rnedanov/golang-pubsub-system/internal/service"
	"github.com/rnedanov/golang-pubsub-system/internal/subpub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	defaultShutdownTimeout = 5 * time.Second
)

type Server struct {
	cfg    *config.Config
	logger *slog.Logger
	grpc   *grpc.Server
	pubsub subpub.SubPub
}

func NewServer(cfg *config.Config) *Server {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return &Server{
		cfg:    cfg,
		logger: logger,
	}
}

func (s *Server) Start() error {
	// Initialize pubsub
	var err error
	s.pubsub, err = subpub.NewSubPub(subpub.ChannelSubPubType, s.cfg)
	if err != nil {
		return fmt.Errorf("failed to create pubsub: %w", err)
	}

	// Create gRPC server
	s.grpc = grpc.NewServer()

	// For testing purposes, we can use reflection to register the server
	// and make it accessible for introspection.
	reflection.Register(s.grpc)

	pubsubService := service.NewPubSubService(s.pubsub, s.logger)

	gen.RegisterPubSubServiceServer(s.grpc, pubsubService)

	// Start gRPC server
	lis, err := net.Listen("tcp", s.cfg.GRPCPort)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.logger.Info("starting gRPC server", "port", s.cfg.GRPCPort)
	return s.grpc.Serve(lis)
}

func (s *Server) Stop() {
	s.logger.Info("shutting down server")
	s.grpc.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()

	if err := s.pubsub.Close(ctx); err != nil {
		s.logger.Error("error closing pubsub", "error", err)
	}
}
