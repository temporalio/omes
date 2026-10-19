package kitchensink

// CreateFiniteScheduleActivityName is implemented by the Go worker.
const CreateFiniteScheduleActivityName = "create_finite_schedule"

// FiniteScheduleInput identifies one finite schedule across activity retries.
type FiniteScheduleInput struct {
	ScheduleID string `json:"scheduleId"`
}
