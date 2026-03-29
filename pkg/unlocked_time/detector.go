package unlocked_time

type IDetector interface {
	IsLocked() bool
	GetUsername() string
}
