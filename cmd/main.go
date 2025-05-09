package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rnedanov/golang-pubsub-system/internal/config"
	"github.com/rnedanov/golang-pubsub-system/internal/subpub"
)

func main() {
	// Init config
	configService, err := config.InitServiceConfig()
	if err != nil {
		log.Fatalf("Error initializing service config: %v", err)
	}

	pubSub, err := subpub.NewSubPub(subpub.ChannelSubPubType, configService)
	if err != nil {
		log.Fatalf("Error creating pubsub system: %v", err)
	}

	// Subscribe with error handling
	sub1, err := pubSub.Subscribe("topic1", func(msg interface{}) {
		fmt.Printf("Subscriber 1 received: %v\n", msg)
		time.Sleep(2 * time.Second)
	})
	if err != nil {
		log.Printf("Error subscribing subscriber 1: %v", err)
	}

	sub2, err := pubSub.Subscribe("topic1", func(msg interface{}) {
		fmt.Printf("Subscriber 2 received: %v\n", msg)
	})
	if err != nil {
		log.Printf("Error subscribing subscriber 2: %v", err)
	}

	// Add some delay to ensure subscribers are ready
	time.Sleep(100 * time.Millisecond)

	// Publish with error handling
	fmt.Println("Publishing Message 1...")
	if err := pubSub.Publish("topic1", "Message 1"); err != nil {
		log.Printf("Error publishing message 1: %v", err)
	}

	fmt.Println("Publishing Message 2...")
	if err := pubSub.Publish("topic1", "Message 2"); err != nil {
		log.Printf("Error publishing message 2: %v", err)
	}

	// Wait for messages to be processed
	time.Sleep(3 * time.Second)

	// Cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if sub1 != nil {
		sub1.Unsubscribe()
	}
	if sub2 != nil {
		sub2.Unsubscribe()
	}

	if err := pubSub.Close(ctx); err != nil {
		fmt.Printf("Error closing pubsub: %v\n", err)
	}
}
