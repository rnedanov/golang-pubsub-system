package subpub

import (
	"context"
	"errors"

	"github.com/rnedanov/golang-pubsub-system/internal/config"
)

const (
	// ChannelSubPubType uses Go channels for pub-sub.
	ChannelSubPubType SubPubType = "channel"
)

// MessageHandler is a callback function that processes messages delivered to subscribers.
type MessageHandler func(msg interface{})

// SubPubType defines the type of SubPub implementation.
type SubPubType string

type Subscription interface {
	// Unsubscribe will remove interest in the current subject subscription is for.
	Unsubscribe()
}

type SubPub interface {
	// Subscribe creates an asynchronous queue subscriber on the given subject.
	Subscribe(subject string, cb MessageHandler) (Subscription, error)

	// Publish publishes the msg argument to the given subject.
	Publish(subject string, msg interface{}) error

	// Close will shutdown sub-pub system.
	// May be blocked by data delivery until the context is canceled.
	Close(ctx context.Context) error
}

func NewSubPub(subPubType SubPubType, configService *config.Config) (SubPub, error) {
	switch subPubType {
	case ChannelSubPubType:
		return NewChannelSubPub(configService.MsgChannelBufSize), nil
	default:
		return nil, errors.New("unknown SubPub type")
	}
}
