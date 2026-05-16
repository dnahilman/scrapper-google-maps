package scraper

import (
	"context"
	"math/rand"
	"time"
)

// HumanDelay sleeps a randomized human-like interval.
// Pass short=true for between-action delays (1-2.5s),
// false for between-request delays (configured min..max seconds).
func HumanDelay(ctx context.Context, minSec, maxSec int, short bool) {
	lo, hi := 1.0, 2.5
	if !short {
		lo = float64(minSec)
		hi = float64(maxSec)
		if hi <= lo {
			hi = lo + 1
		}
	}
	d := time.Duration((lo + rand.Float64()*(hi-lo)) * float64(time.Second))
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
