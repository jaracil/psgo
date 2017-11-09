package psgo

import (
	"strings"
	"sync"
)

type Msg struct {
	To  string
	Res string
	Dat interface{}
	Old bool
}

type MsgOpts struct {
	Persist bool
	NoRec   bool
}

type Subscriber struct {
	subs map[string]bool
	f    func(msg *Msg)
	lock sync.Mutex
}

var subscriptions = map[string]map[*Subscriber]bool{}
var oldMessages = map[string]*Msg{}
var psLock sync.Mutex

func NewSubscriber(f func(msg *Msg)) *Subscriber {
	return &Subscriber{subs: map[string]bool{}, f: f}
}

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

func (su *Subscriber) UnsubscribeAll() {
	su.Unsubscribe(su.Subscriptions()...)
}

func (su *Subscriber) Subscriptions() (ret []string) {
	su.lock.Lock()
	defer su.lock.Unlock()
	for key := range su.subs {
		ret = append(ret, key)
	}
	return
}

func (su *Subscriber) NumSubscriptions() int {
	su.lock.Lock()
	defer su.lock.Unlock()
	return len(su.subs)
}

func NumSubscribers(path string) int {
	psLock.Lock()
	defer psLock.Unlock()
	return len(subscriptions[path])
}

func Publish(msg *Msg, opts ...*MsgOpts) (cnt int) {
	var op *MsgOpts
	if len(opts) > 0 {
		op = opts[0]
	} else {
		op = &MsgOpts{} // Defaul options
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
				go su.f(msg)
			}
		}
		if op.NoRec {
			break
		}
		chunks = chunks[:len(chunks)-1]
	}
	return
}
