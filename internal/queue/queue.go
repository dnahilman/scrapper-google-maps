package queue

import (
	"context"

	"github.com/google/uuid"
)

// ClaimedTask is a denormalized view returned to the worker on dequeue.
type ClaimedTask struct {
	TaskID          uuid.UUID `json:"task_id"`
	JobID           uuid.UUID `json:"job_id"`
	Keyword         string    `json:"keyword"`
	KelurahanID     uuid.UUID `json:"kelurahan_id"`
	KelurahanName   string    `json:"kelurahan_name"`
	KecamatanName   string    `json:"kecamatan_name"`
	CityName        string    `json:"city_name"`
	Attempt         int       `json:"attempt"`
	MaxAttempts     int       `json:"max_attempts"`
	Options         []byte    `json:"options,omitempty"` // raw JSONB
}

// Queue is the master-side job queue interface.
type Queue interface {
	// Claim atomically dequeues one task for the given worker (returns nil if empty).
	Claim(ctx context.Context, workerID uuid.UUID) (*ClaimedTask, error)

	// Ack marks the task done.
	Ack(ctx context.Context, taskID uuid.UUID, placesCount int, resultPath string) error

	// Nack marks the task failed (re-queued with backoff if attempts remain).
	Nack(ctx context.Context, taskID uuid.UUID, errMsg string) error

	// Heartbeat refreshes liveness for a running task.
	Heartbeat(ctx context.Context, taskID uuid.UUID) error
}

// Statically assert PostgresQueue implements Queue.
var _ Queue = (*PostgresQueue)(nil)
