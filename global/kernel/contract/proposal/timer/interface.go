package timer

type TimerManager interface {
	GetTimerTasks(blockHeight int64) (uint64, error)
}
