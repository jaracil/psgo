// Package psgo is a lightweight implementation of pub/sub paradigm.
package psgo

import (
	"strings"
	"sync"
)

// Msg is the message sent to subscribers
type Msg struct {
	To  string      // Target path i.e. "system.status"
	Res string      // Optional response path.
	Dat interface{} // Message payload
	Old bool        // This message is old (See Persist option in MsgOpts)
}

func (msg Msg) Answer(dat interface{}) {
	Publish(&Msg{To: msg.Res, Dat: dat}, &MsgOpts{NoPropagate: true})
}

// MsgOpts contains optional message flags
type MsgOpts struct {
	Persist     bool // New subscribers will receive the last message sent in subscription path
	NoPropagate bool // No propagate message to path ancestors
	Sync        bool // If false executes each callback in its own goroutine
}

// Subscriber is the type that tracks subscriptions
type Subscriber struct {
	subs map[string]bool
	f    func(msg *Msg)
	lock sync.Mutex
}

var subscriptions = map[string]map[*Subscriber]bool{}
var oldMessages = map[string]*Msg{}
var psLock sync.Mutex

// NewSubscriber creates new initialized subscriber
//
// f param is the function called when message arrives
func NewSubscriber(f func(msg *Msg)) *Subscriber {
	return &Subscriber{subs: map[string]bool{}, f: f}
}

// Subscribe adds subscriptions to subscriber
//
// i.e. subscriber.Subscribe("foo.bar", "fizz.buzz")
func (su *Subscriber) Subscribe(paths ...string) {
	su.lock.Lock()
	defer su.lock.Unlock()
	psLock.Lock()
	defer psLock.Unlock()
	for _, path := range paths {
		su.subs[path] = true
		if subscriptions[path] == nil {
			subscriptions[path] = map[*Subscriber]bool{}
		}
		subscriptions[path][su] = true
		msg := oldMessages[path]
		if msg != nil {
			go su.f(msg)
		}
	}
	return
}

// Unsubscribe removes subscriptions from subscriber
//
// i.e. subscriber.Unsubscribe("foo.bar", "fizz.buzz")
func (su *Subscriber) Unsubscribe(paths ...string) {
	su.lock.Lock()
	defer su.lock.Unlock()
	psLock.Lock()
	defer psLock.Unlock()
	for _, path := range paths {
		delete(su.subs, path)
		subs := subscriptions[path]
		if subs != nil {
			delete(subs, su)
			if len(subs) == 0 {
				delete(subscriptions, path)
			}
		}
	}
	return
}

// UnsubscribeAll removes all subscriptions
func (su *Subscriber) UnsubscribeAll() {
	su.Unsubscribe(su.Subscriptions()...)
}

// Subscriptions returns a slice with all subscription paths
func (su *Subscriber) Subscriptions() (ret []string) {
	su.lock.Lock()
	defer su.lock.Unlock()
	for key := range su.subs {
		ret = append(ret, key)
	}
	return
}

// NumSubscriptions returns the number of subscriber alive subscriptions
func (su *Subscriber) NumSubscriptions() int {
	su.lock.Lock()
	defer su.lock.Unlock()
	return len(su.subs)
}

// NumSubscribers returns the number of subscribers attached to the path
func NumSubscribers(path string) int {
	psLock.Lock()
	defer psLock.Unlock()
	return len(subscriptions[path])
}

// Publish sends the message to all subscriptors attached to the message tartget path (Msg.To).
//
// If MsgOpts.NoPropagate is true, msg is not sent to subscribers attached to ancestor paths.
// i.e. Publish(&psgo.Msg{To:"A.B", Dat:"foo"}, &psgo.MsgOpts{NoPropagate: true}) will send message
// to subscribers attached to "A.B" but no "A"
func Publish(msg *Msg, opts ...*MsgOpts) (cnt int) {
	var op *MsgOpts
	if len(opts) > 0 {
		op = opts[0]
	} else {
		op = &MsgOpts{} // Default options
	}
	psLock.Lock()
	defer psLock.Unlock()
	if op.Persist {
		msgCpy := *msg
		msgCpy.Old = true
		oldMessages[msgCpy.To] = &msgCpy
	}
	chunks := strings.Split(msg.To, ".")
	for len(chunks) > 0 {
		key := strings.Join(chunks, ".")
		subs := subscriptions[key]
		if subs != nil {
			for su := range subs {
				cnt++
				if op.Sync {
					su.f(msg)
				} else {
					go su.f(msg)
				}
			}
		}
		if op.NoPropagate {
			break
		}
		chunks = chunks[:len(chunks)-1]
	}
	return
}
