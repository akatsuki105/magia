package scheduler

type EventName string

const (
	StartHBlank  EventName = "StartHBlank"
	MidHBlank    EventName = "MidHBlank"
	StartHDraw   EventName = "StartHDraw"
	Timer0Update EventName = "Timer0Update"
	Timer1Update EventName = "Timer1Update"
	Timer2Update EventName = "Timer2Update"
	Timer3Update EventName = "Timer3Update"
	Irq          EventName = "Irq"
)

type Event struct {
	name     EventName
	callback func(cyclesLate uint64)
	when     uint64
	next     *Event
}
