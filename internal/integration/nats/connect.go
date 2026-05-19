package nats

import (
	"context"
	"fmt"
	"strings"

	natsc "github.com/nats-io/nats.go"
)

// ConnectPublisher dials NATS and returns a drain close hook plus a Publisher for outbound events.
func ConnectPublisher(rawURL string) (close func(), pub Publisher, err error) {
	u := strings.TrimSpace(rawURL)
	if u == "" {
		return nil, nil, fmt.Errorf("nats: url is required")
	}
	nc, err := natsc.Connect(u)
	if err != nil {
		return nil, nil, fmt.Errorf("nats connect: %w", err)
	}
	return func() { _ = nc.Drain() }, natsConnPublisher{nc: nc}, nil
}

type natsConnPublisher struct {
	nc *natsc.Conn
}

func (p natsConnPublisher) Publish(_ context.Context, subject string, payload []byte) error {
	return p.nc.Publish(subject, payload)
}
