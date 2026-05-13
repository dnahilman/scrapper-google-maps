package workeragent

import (
	"context"
	"time"

	"github.com/dnahilman/scrapper-go/internal/domain"
	"github.com/dnahilman/scrapper-go/internal/queue"
)

// NoopExecutor simulates work for Phase 1 smoke tests.
// It sleeps briefly and returns zero places.
type NoopExecutor struct {
	Delay time.Duration
}

func (n *NoopExecutor) Execute(ctx context.Context, _ *queue.ClaimedTask) ([]domain.PlacePayload, error) {
	d := n.Delay
	if d <= 0 {
		d = 2 * time.Second
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(d):
	}
	return nil, nil
}
