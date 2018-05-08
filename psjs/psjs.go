package psjs

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/gopherjs/gopherjs/js"
	"github.com/jaracil/psgo"
)

type Msg struct {
	*js.Object
	To  string      `js:"to"`
	Res string      `js:"res"`
	Dat interface{} `js:"dat"`
	Old bool        `js:"old"`
}

type MsgOpts struct {
	*js.Object
	Persist     bool `js:"persist"`
	NoPropagate bool `js:"noPropagate"`
	Sync        bool `js:"sync"`
}

type Promise struct {
	*js.Object
	Done         bool
	ThenPending  bool
	CatchPending bool
	Then         func(*psgo.Msg)
	Catch        func(error)
	Answer       *psgo.Msg
}

var idCnt = 0
var subs = map[int]*psgo.Subscriber{}
var timeoutError = errors.New("ERROR_TIMEOUT")

func init() {
	ob := js.Global.Get("Object").New()
	js.Global.Set("psgo", ob)
	ob.Set("newSubscriber", newSubscriber)
	ob.Set("subscribe", subscribe)
	ob.Set("unsubscribe", unsubscribe)
	ob.Set("unsubscribeAll", unsubscribeAll)
	ob.Set("numSubscriptions", numSubscriptions)
	ob.Set("subscriptions", subscriptions)
	ob.Set("close", close)
	ob.Set("publish", publish)
	ob.Set("numSubscribers", numSubscribers)
	ob.Set("call", call)
}

func newSubscriber(f func(m *Msg)) int {
	wf := func(m *psgo.Msg) {
		wm := &Msg{Object: js.Global.Get("Object").New()}
		wm.To = m.To
		wm.Res = m.Res
		wm.Dat = m.Dat
		wm.Old = m.Old
		f(wm)

	}
	idCnt++
	subs[idCnt] = psgo.NewSubscriber(wf)
	return idCnt
}

func subscribe(id int, paths ...string) {
	subs[id].Subscribe(paths...)
}

func unsubscribe(id int, paths ...string) {
	subs[id].Unsubscribe(paths...)
}

func unsubscribeAll(id int) {
	subs[id].UnsubscribeAll()
}

func numSubscriptions(id int) int {
	return subs[id].NumSubscriptions()
}

func subscriptions(id int) []string {
	return subs[id].Subscriptions()
}

func close(id int) {
	unsubscribeAll(id)
	delete(subs, id)
}

func publish(msg *Msg, opts ...*MsgOpts) int {
	m := &psgo.Msg{To: msg.To, Res: msg.Res, Dat: msg.Dat}
	if len(opts) > 0 {
		opt := opts[0]
		o := &psgo.MsgOpts{Persist: opt.Persist, NoPropagate: opt.NoPropagate, Sync: opt.Sync}
		return psgo.Publish(m, o)
	}
	return psgo.Publish(m)
}

func numSubscribers(path string) int {
	return psgo.NumSubscribers(path)
}

func call(path string, value interface{}, timeout int64) *Promise {
	prom := &Promise{Object: js.Global.Get("Object").New()}
	prom.Set("then", func(a func(msg *psgo.Msg)) *Promise {
		prom.Then = a
		if !prom.Done && prom.ThenPending {
			prom.Then(prom.Answer)
		}
		return prom
	})
	prom.Set("catch", func(a func(error)) *Promise {
		prom.Catch = a
		if !prom.Done && prom.CatchPending {
			prom.Catch(timeoutError)
		}
		return prom
	})

	token := createUniquePath()

	timeoutTimer := time.AfterFunc(time.Duration(timeout)*time.Second, func() {
		if prom.Catch == nil {
			prom.CatchPending = true
		} else {
			prom.Catch(timeoutError)
		}
	})

	go func(prom *Promise, timeoutTimer *time.Timer) {

		var subscriber *psgo.Subscriber
		subscriber = psgo.NewSubscriber(func(msg *psgo.Msg) {
			timeoutTimer.Stop()

			if prom.Then == nil {
				prom.ThenPending = true
				prom.Answer = msg
			} else {
				prom.Then(msg)
			}

			subscriber.Unsubscribe(token)
		})
		subscriber.Subscribe(token)

		psgo.Publish(&psgo.Msg{To: path, Dat: value, Res: token})
	}(prom, timeoutTimer)

	return prom
}

func createUniquePath() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("calls.%x", b)
}
