package gallagbus

import (
	"fmt"
	"reflect"
)

type EventListener struct {
	callback reflect.Value
	queue    chan []reflect.Value
}

type EventListenerOptions func(*EventListener)

func QueueSize(size int) EventListenerOptions {
	return func(listener *EventListener) {
		listener.queue = make(chan []reflect.Value, size)
	}
}

func NewEventListener(fn interface{}, opts ...EventListenerOptions) EventListener {
	if reflect.TypeOf(fn).Kind() != reflect.Func {
		panic(fmt.Sprintf("%s is not a reflect.Func", reflect.TypeOf(fn)))
	}

	h := EventListener{
		callback: reflect.ValueOf(fn),
		queue:    make(chan []reflect.Value, 10),
	}

	for _, opt := range opts {
		opt(&h)
	}

	go func() {
		for args := range h.queue {
			h.Call(args)
		}
	}()

	return h
}

func (e EventListener) ExpectedEvent() reflect.Type {
	if e.callback.Type().NumIn() > 0 {
		return e.callback.Type().In(0)
	}
	return reflect.Type(nil)
}

func (e EventListener) Call(values []reflect.Value) {
	e.callback.Call(values)
}

type EventBus interface {
	Publish(eventName string, event interface{})
	Subscribe(eventName string, eventListener EventListener)
}

type GallagBus struct {
	eventListeners map[string][]EventListener
}

var _ EventBus = &GallagBus{}

// This closes every channel on every handler
func (g GallagBus) GracefulShutdown() {
	for _, listeners := range g.eventListeners {
		for _, l := range listeners {
			close(l.queue)
		}
	}
}

func New() *GallagBus {
	listeners := make(map[string][]EventListener)
	return &GallagBus{eventListeners: listeners}
}

// Publish an Event
func (g *GallagBus) Publish(eventType string, event interface{}) {
	if hs, ok := g.eventListeners[eventType]; ok {
		eValue := reflect.ValueOf(event)
		eType := reflect.TypeOf(event)
		values := []reflect.Value{eValue}
		for _, h := range hs {
			if h.ExpectedEvent() == eType {
				h.queue <- values
			}
		}
	}
}

func (g *GallagBus) Subscribe(eventType string, eventListener EventListener) {
	g.eventListeners[eventType] = append(g.eventListeners[eventType], eventListener)
}
