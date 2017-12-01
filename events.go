package main

import "time"

type EventType int

const (
	EventLog EventType = 1
	EventScan EventType = 2
	EventDashChanged EventType = 3
	EventInvChanged EventType = 4
)

type Event struct {
	Type EventType
	Attachment interface{}
}

func LogEvent(msg string) Event {
	return Event{EventLog, msg}
}

func ScanEvent() Event {
	return Event{EventScan, time.Now()}
}

func DashChangedEvent() Event {
	return Event{EventDashChanged, nil}
}

func InvChangedEvent() Event {
	return Event{EventInvChanged, nil}
}
