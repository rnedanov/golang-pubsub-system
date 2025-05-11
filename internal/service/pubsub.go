package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rnedanov/golang-pubsub-system/gen"
	"github.com/rnedanov/golang-pubsub-system/internal/subpub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type PubSubService struct {
	gen.UnimplementedPubSubServiceServer
	pubsub subpub.SubPub
	logger *slog.Logger
}

func NewPubSubService(ps subpub.SubPub, logger *slog.Logger) *PubSubService {
	return &PubSubService{
		pubsub: ps,
		logger: logger,
	}
}

func (s *PubSubService) Subscribe(req *gen.SubscribeRequest, stream gen.PubSubService_SubscribeServer) error {
	if req.Key == "" {
		return status.Error(codes.InvalidArgument, "key cannot be empty")
	}

	s.logger.Info("new subscription request", "key", req.Key)

	// Create channel for handling stream closure
	done := make(chan struct{})
	defer close(done)

	// Create error channel to propagate errors from callback
	errChan := make(chan error, 1)

	// Subscribe to events
	sub, err := s.pubsub.Subscribe(req.Key, func(msg interface{}) {
		str, ok := msg.(string)
		if !ok {
			s.logger.Error("invalid message type", "type", fmt.Sprintf("%T", msg))
			errChan <- status.Error(codes.Internal, "received message of invalid type")
			return
		}

		event := &gen.Event{Data: str}
		if err := stream.Send(event); err != nil {
			s.logger.Error("failed to send event", "error", err)
			errChan <- status.Error(codes.Internal, "failed to send event")
			return
		}
	})
	if err != nil {
		s.logger.Error("failed to subscribe", "error", err)
		return status.Error(codes.Internal, "failed to subscribe")
	}
	defer sub.Unsubscribe()

	// Wait for either context cancellation or error
	select {
	case <-stream.Context().Done():
		s.logger.Info("subscription ended", "key", req.Key)
		return nil
	case err := <-errChan:
		s.logger.Error("subscription failed", "key", req.Key, "error", err)
		return err
	}
}

func (s *PubSubService) Publish(ctx context.Context, req *gen.PublishRequest) (*emptypb.Empty, error) {
	if req.Key == "" {
		return nil, status.Error(codes.InvalidArgument, "key cannot be empty")
	}

	s.logger.Info("publishing message", "key", req.Key, "data", req.Data)

	if err := s.pubsub.Publish(req.Key, req.Data); err != nil {
		s.logger.Error("failed to publish", "error", err)
		return nil, status.Error(codes.Internal, "failed to publish message")
	}

	return &emptypb.Empty{}, nil
}
