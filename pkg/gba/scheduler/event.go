package scheduler

type EventName string

const ()

type Event struct {
	name     EventName
	callback func(cyclesLate uint64)
	when     uint64
	next     *Event
}
