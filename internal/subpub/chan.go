package subpub

import (
	"context"
	"errors"
	"log"
	"slices"
	"sync"
)

type channelSubPub struct {
	mu          sync.RWMutex
	subscribers map[string][]chan interface{}
	closed      bool
	wg          sync.WaitGroup
	bufferSize  int
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewChannelSubPub(bufferSize int) SubPub {
	ctx, cancel := context.WithCancel(context.Background())
	return &channelSubPub{
		subscribers: make(map[string][]chan interface{}),
		bufferSize:  bufferSize,
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (c *channelSubPub) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, errors.New("subpub system is closed")
	}

	// Create a buffered channel for the subscriber
	msgChan := make(chan interface{}, c.bufferSize)
	c.subscribers[subject] = append(c.subscribers[subject], msgChan)

	// Start a goroutine to receive messages
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case msg, ok := <-msgChan:
				if !ok {
					return
				}
				// Process messages synchronously to maintain order
				cb(msg)
			case <-c.ctx.Done():
				return
			}
		}
	}()

	return &channelSubscription{
		subject: subject,
		msgChan: msgChan,
		parent:  c,
	}, nil
}

func (c *channelSubPub) Publish(subject string, msg interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return errors.New("subpub system is closed")
	}

	if chans, ok := c.subscribers[subject]; ok {
		log.Printf("Publishing message '%v' to %d subscribers for subject '%s'", msg, len(chans), subject)
		// Send message to all subscribers synchronously to maintain order
		for _, ch := range chans {
			select {
			case ch <- msg:
				// Message sent successfully
			case <-c.ctx.Done():
				return c.ctx.Err()
			}
		}
	}

	return nil
}

func (c *channelSubPub) Close(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return errors.New("subpub system is already closed")
	}
	c.closed = true

	// Cancel all goroutines
	c.cancel()

	// Close all subscriber channels
	for _, chans := range c.subscribers {
		for _, ch := range chans {
			close(ch)
		}
	}
	c.subscribers = nil
	c.mu.Unlock()

	// Wait for all goroutines to complete or context to be canceled
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

type channelSubscription struct {
	subject string
	msgChan chan interface{}
	parent  *channelSubPub
}

func (s *channelSubscription) Unsubscribe() {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	if s.parent.closed {
		return
	}

	chans := s.parent.subscribers[s.subject]
	for i, ch := range chans {
		if ch == s.msgChan {
			// Remove channel from slice
			s.parent.subscribers[s.subject] = slices.Delete(chans, i, i+1)
			close(ch)
			break
		}
	}
}
