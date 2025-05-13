# PubSub gRPC Service

This service implements a publish-subscribe system over gRPC using the internal `subpub` package.

## Features
- Subscribe to events by key
- Publish events to all subscribers
- Configurable through environment variables
- Graceful shutdown
- Structured logging
- GRPC status codes for error handling

## Configuration
The service can be configured through environment variables or a `.env` file:

```env
GRPC_PORT=:50051                # gRPC server port
MSG_CHANNEL_BUFFER_SIZE=100     # Buffer size for message channels
LOG_LEVEL=info                  # Logging level (debug, info, warn, error)
```

## Building and Running
1. Generate protobuf files:
```bash {: .copy}
protoc --proto_path=protos \
       --go_out=./gen/ \
       --go_opt=paths=source_relative \
       --go-grpc_out=./gen/ \
       --go-grpc_opt=paths=source_relative \
       protos/pubsub.proto
```

2. Build the service:
```bash {: .copy}
go build -o pubsub-service ./cmd/main.go
```

3. Run the service:
```bash {: .copy}
./pubsub-service
```

## Design Patterns Used
1. Dependency Injection
   - Services receive their dependencies through constructor parameters
   - Makes testing and maintenance easier

2. Graceful Shutdown
   - Handles SIGINT and SIGTERM signals
   - Allows in-flight requests to complete
   - Properly closes all resources

3. Configuration Management
   - External configuration through environment variables
   - Sensible defaults for all settings

4. Structured Logging
   - Uses slog for structured, leveled logging
   - Consistent log format across the service

## Testing
Run all tests with coverage:
```bash {: .copy}
go test -v -cover ./...
```

Run specific package tests:
```bash {: .copy}
# Run subpub package tests
go test -v ./internal/subpub

# Run service tests
go test -v ./internal/service
```

## Usage Example

### Using Go Client
```go
// Client code example
conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
client := gen.NewPubSubServiceClient(conn)

// Subscribe
stream, err := client.Subscribe(&gen.SubscribeRequest{Key: "mykey"})
go func() {
    for {
        event, err := stream.Recv()
        if err != nil {
            log.Printf("Stream ended: %v", err)
            return
        }
        log.Printf("Received: %v", event.Data)
    }
}()

// Publish
_, err = client.Publish(context.Background(), &gen.PublishRequest{
    Key:  "mykey",
    Data: "Hello, World!",
})
```

### Using gRPCurl
First, install gRPCurl:
```bash {: .copy}
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

List available services:
```bash {: .copy}
grpcurl -plaintext localhost:50051 list
```

Subscribe to events (in a separate terminal):
```bash {: .copy}
grpcurl -plaintext -d '{"key": "test-key"}' localhost:50051 main.PubSubService/Subscribe
```

Publish a message:
```bash {: .copy}
grpcurl -plaintext -d '{"key": "test-key", "data": "Hello from gRPCurl!"}' \
    localhost:50051 main.PubSubService/Publish
```

The subscriber terminal should receive the message:
```json
{
  "data": "Hello from gRPCurl!"
}
```

Note: The service uses reflection, which allows gRPCurl to discover the service methods automatically.