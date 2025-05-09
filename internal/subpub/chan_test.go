package subpub

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestChannelSubPub_MultipleSubscribers(t *testing.T) {
	pubsub := NewChannelSubPub(10)

	var wg sync.WaitGroup
	messages1 := make([]interface{}, 0)
	messages2 := make([]interface{}, 0)
	var mu sync.Mutex

	// Subscribe two subscribers to one subject
	sub1, err := pubsub.Subscribe("test", func(msg interface{}) {
		mu.Lock()
		messages1 = append(messages1, msg)
		mu.Unlock()
		wg.Done()
	})
	assert.NoError(t, err)
	assert.NotNil(t, sub1)

	sub2, err := pubsub.Subscribe("test", func(msg interface{}) {
		mu.Lock()
		messages2 = append(messages2, msg)
		mu.Unlock()
		wg.Done()
	})
	assert.NoError(t, err)
	assert.NotNil(t, sub2)

	// Publish messages
	wg.Add(4) // 2 messages * 2 subscribers
	assert.NoError(t, pubsub.Publish("test", "msg1"))
	assert.NoError(t, pubsub.Publish("test", "msg2"))

	wg.Wait()

	// Check that both subscribers received all messages in correct order
	assert.Equal(t, []interface{}{"msg1", "msg2"}, messages1)
	assert.Equal(t, []interface{}{"msg1", "msg2"}, messages2)
}

func TestChannelSubPub_SlowSubscriber(t *testing.T) {
	pubsub := NewChannelSubPub(10)

	var wg sync.WaitGroup
	var fastWg sync.WaitGroup
	messages1 := make([]interface{}, 0)
	messages2 := make([]interface{}, 0)
	var mu sync.Mutex

	// Slow subscriber
	slowSub, err := pubsub.Subscribe("test", func(msg interface{}) {
		time.Sleep(200 * time.Millisecond) // Simulate slow processing
		mu.Lock()
		messages1 = append(messages1, msg)
		mu.Unlock()
		wg.Done()
	})
	assert.NoError(t, err)
	defer slowSub.Unsubscribe()

	// Fast subscriber
	fastSub, err := pubsub.Subscribe("test", func(msg interface{}) {
		mu.Lock()
		messages2 = append(messages2, msg)
		mu.Unlock()
		fastWg.Done()
	})
	assert.NoError(t, err)
	defer fastSub.Unsubscribe()

	// Publish messages
	wg.Add(2)     // For slow subscriber
	fastWg.Add(2) // For fast subscriber
	start := time.Now()

	assert.NoError(t, pubsub.Publish("test", "msg1"))
	assert.NoError(t, pubsub.Publish("test", "msg2"))

	// Wait only for fast subscriber
	fastWg.Wait()
	duration := time.Since(start)

	// Check that fast subscriber processed messages quickly
	assert.Less(t, duration, 100*time.Millisecond, "Fast subscriber should process messages quickly")
	assert.Equal(t, []interface{}{"msg1", "msg2"}, messages2, "Fast subscriber should maintain message order")

	// Wait for slow subscriber
	wg.Wait()
	assert.Equal(t, []interface{}{"msg1", "msg2"}, messages1, "Slow subscriber should maintain message order")
}

func TestChannelSubPub_Close(t *testing.T) {
	pubsub := NewChannelSubPub(10)

	var wg sync.WaitGroup
	processed := make(chan struct{})

	// Subscribe a handler that will work for a long time
	_, err := pubsub.Subscribe("test", func(msg interface{}) {
		defer wg.Done()
		time.Sleep(200 * time.Millisecond)
	})
	assert.NoError(t, err)

	// Publish message
	wg.Add(1)
	assert.NoError(t, pubsub.Publish("test", "msg"))

	// Close system with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go func() {
		wg.Wait()
		close(processed)
	}()

	// Check that Close returns context error
	err = pubsub.Close(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)

	// Check that handler still completed its work
	<-processed
}

func TestChannelSubPub_Unsubscribe(t *testing.T) {
	pubsub := NewChannelSubPub(10)

	var wg sync.WaitGroup
	messages := make([]interface{}, 0)
	var mu sync.Mutex

	sub, err := pubsub.Subscribe("test", func(msg interface{}) {
		mu.Lock()
		messages = append(messages, msg)
		mu.Unlock()
		wg.Done()
	})
	assert.NoError(t, err)

	// Publish first message
	wg.Add(1)
	assert.NoError(t, pubsub.Publish("test", "msg1"))
	wg.Wait()

	// Unsubscribe
	sub.Unsubscribe()

	// Publish second message
	assert.NoError(t, pubsub.Publish("test", "msg2"))
	time.Sleep(100 * time.Millisecond)

	// Check that only first message was received
	assert.Equal(t, []interface{}{"msg1"}, messages)
}
