package nats

import "context"

// Publisher publishes raw NATS messages (assessment outbound / request publish).
type Publisher interface {
	Publish(ctx context.Context, subject string, payload []byte) error
}
