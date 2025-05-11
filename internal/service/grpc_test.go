package service_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/rnedanov/golang-pubsub-system/gen"
	"github.com/rnedanov/golang-pubsub-system/internal/service"
	"github.com/rnedanov/golang-pubsub-system/internal/subpub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func setupTestServer(t *testing.T) (*grpc.ClientConn, func()) {
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()

	pubsub := subpub.NewChannelSubPub(10)

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	gen.RegisterPubSubServiceServer(s, service.NewPubSubService(pubsub, logger))

	go func() {
		if err := s.Serve(lis); err != nil {
			t.Logf("Server exited with error: %v", err)
		}
	}()

	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithInsecure(),
	)
	require.NoError(t, err)

	return conn, func() {
		conn.Close()
		s.Stop()
	}
}

func TestPublish_ValidRequest(t *testing.T) {
	conn, closer := setupTestServer(t)
	defer closer()
	client := gen.NewPubSubServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.Publish(ctx, &gen.PublishRequest{
		Key:  "test",
		Data: "message",
	})
	assert.NoError(t, err)
}

func TestPublish_InvalidRequest(t *testing.T) {
	conn, closer := setupTestServer(t)
	defer closer()
	client := gen.NewPubSubServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.Publish(ctx, &gen.PublishRequest{Key: ""})
	assert.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSubscribe_StreamMessages(t *testing.T) {
	conn, closer := setupTestServer(t)
	defer closer()
	client := gen.NewPubSubServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stream, err := client.Subscribe(ctx, &gen.SubscribeRequest{Key: "test"})
	require.NoError(t, err)

	// Publish messages in background
	go func() {
		time.Sleep(100 * time.Millisecond)
		client.Publish(context.Background(), &gen.PublishRequest{
			Key:  "test",
			Data: "msg1",
		})
		client.Publish(context.Background(), &gen.PublishRequest{
			Key:  "test",
			Data: "msg2",
		})
	}()

	var received []string
	for i := 0; i < 2; i++ {
		event, err := stream.Recv()
		if err != nil {
			break
		}
		received = append(received, event.Data)
	}

	assert.Equal(t, []string{"msg1", "msg2"}, received)
}

func TestSubscribe_ContextCancel(t *testing.T) {
	conn, closer := setupTestServer(t)
	defer closer()
	client := gen.NewPubSubServiceClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	stream, err := client.Subscribe(ctx, &gen.SubscribeRequest{Key: "test"})
	require.NoError(t, err)

	// Cancel context and verify stream closure
	cancel()

	_, err = stream.Recv()
	assert.Error(t, err)
	assert.Equal(t, codes.Canceled, status.Code(err))
}

func TestService_UnsupportedMessageType(t *testing.T) {
	pubsub := subpub.NewChannelSubPub(10)
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	svc := service.NewPubSubService(pubsub, logger)

	// Create context with timeout to control test execution time
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Mock stream with updated context
	stream := &mockSubscribeServer{ctx: ctx}

	// Run Subscribe asynchronously as it blocks waiting for messages
	errChan := make(chan error, 1)
	go func() {
		errChan <- svc.Subscribe(&gen.SubscribeRequest{Key: "test"}, stream)
	}()

	// Allow time for subscription setup
	time.Sleep(100 * time.Millisecond)

	// Publish message with invalid type (int instead of string)
	err := pubsub.Publish("test", 123)
	require.NoError(t, err)

	// Wait for result with timeout
	select {
	case err := <-errChan:
		assert.Error(t, err)
		assert.Equal(t, codes.Internal, status.Code(err))
	case <-ctx.Done():
		t.Fatal("Test timeout: no error received within the allocated time")
	}
}

// Mock implementation for server stream
type mockSubscribeServer struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockSubscribeServer) Context() context.Context {
	return m.ctx
}

func (m *mockSubscribeServer) Send(*gen.Event) error {
	return nil
}
