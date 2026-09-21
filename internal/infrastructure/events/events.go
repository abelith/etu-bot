package events

import (
	"context"
	"sync"
)

type Event map[string]any

type EventLog interface {
	Write(event Event)
}

var Log EventLog = &noOpLog{}
var mu sync.Mutex

func SetLogger(log EventLog) {
	mu.Lock()
	Log = log
	mu.Unlock()
}

func FromContext(ctx context.Context) EventLog {
	return ctx.Value(eventLogKey{}).(EventLog)
}

func With(ctx context.Context, log EventLog) context.Context {
	childCtx := context.WithValue(ctx, eventLogKey{}, log)
	return childCtx
}

type eventLogKey struct{}

type noOpLog struct{}

func (*noOpLog) Write(_ Event) {
	return
}
