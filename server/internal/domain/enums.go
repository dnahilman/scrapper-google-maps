package domain

type WorkerStatus string

const (
	WorkerStatusOnline    WorkerStatus = "online"
	WorkerStatusOffline   WorkerStatus = "offline"
	WorkerStatusDraining  WorkerStatus = "draining"
)

type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

type TaskStatus string

const (
	TaskStatusQueued     TaskStatus = "queued"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

type PlaceStatus string

const (
	PlaceStatusActive             PlaceStatus = "active"
	PlaceStatusTemporarilyClosed  PlaceStatus = "temporarily_closed"
	PlaceStatusPermanentlyClosed  PlaceStatus = "permanently_closed"
)
