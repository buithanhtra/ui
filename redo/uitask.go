// 6 july 2014

package ui

import (
	"runtime"
	"sync"
)

// Go initializes package ui.
// TODO write this bit
func Go() error {
	runtime.LockOSThread()
	if err := uiinit(); err != nil {
		return err
	}
	go uitask()
	uimsgloop()
	return nil
}

// Stop returns a Request for package ui to stop.
// Some time after this request is received, Go() will return without performing any final cleanup.
// If Stop is issued during an event handler, it will be registered when the event handler returns.
func Stop() *Request {
	c := make(chan interface{})
	return &Request{
		op:		func() {
			uistop()
			c <- struct{}{}
		},
		resp:		c,
	}
}

// This is the ui main loop.
// It is spawned by Go as a goroutine.
func uitask() {
	for {
		select {
		case req := <-Do:
			// TODO foreign event
			issue(req)
		case <-stall:		// wait for event to finish
			<-stall		// see below for information
		}
	}
}

// At each event, this is pulsed twice: once when the event begins, and once when the event ends.
// Do is not processed in between.
var stall = make(chan struct{})

type event struct {
	// All events internally return bool; those that don't will be wrapped around to return a dummy value.
	do		func(c Doer) bool
	lock		sync.Mutex
}

// do should never be nil; TODO should we make setters panic instead?

func newEvent() *event {
	return &event{
		do:	func(c Doer) bool {
			return false
		},
	}
}

func (e *event) set(f func(Doer)) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if f == nil {
		f = func(c Doer) {}
	}
	e.do = func(c Doer) bool {
		f(c)
		return false
	}
}

func (e *event) setbool(f func(Doer) bool) {
	e.lock.Lock()
	defer e.lock.Unlock()

	if f == nil {
		f = func(c Doer) bool {
			return false
		}
	}
	e.do = f
}

// This is the common code for running an event.
// It runs on the main thread without a message pump; it provides its own.
func (e *event) fire() bool {
	stall <- struct{}{}		// enter event handler
	c := make(Doer)
	result := false
	go func() {
		e.lock.Lock()
		defer e.lock.Unlock()

		result = e.do(c)
		close(c)
	}()
	for req := range c {
		// note: this is perform, not issue!
		// doevent runs on the main thread without a message pump!
		perform(req)
	}
	// leave the event handler; leave it only after returning from an event handler so we must issue it like a normal Request
	issue(&Request{
		op:		func() {
			stall <- struct{}{}
		},
		// unfortunately, closing a nil channel causes a panic
		// therefore, we have to make a dummy channel
		// TODO add conditional checks to the request handler instead?
		resp:		make(chan interface{}),
	})
	return result
}

// Common code for performing a Request.
// This should run on the main thread.
// Implementations of issue() should call this.
func perform(req *Request) {
	req.op()
	close(req.resp)
}
