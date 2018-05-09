package psjs

import (
	"context"
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
	State  string
	Then   func(interface{})
	Catch  func(error)
	Answer interface{}
}

var idCnt = 0
var subs = map[int]*psgo.Subscriber{}
var timeoutError = errors.New("ERROR_TIMEOUT")

const RESOLVE_STATE = "resolve"
const REJECT_STATE = "reject"

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
	prom.Set("then", func(a func(interface{})) *Promise {
		prom.Then = a
		if prom.State == RESOLVE_STATE {
			prom.Then(prom.Answer)
		}
		return prom
	})
	prom.Set("catch", func(a func(error)) *Promise {
		prom.Catch = a
		if prom.State == REJECT_STATE {
			prom.Catch(timeoutError)
		}
		return prom
	})

	ctx := context.Background()
	var canFunc context.CancelFunc
	if timeout > 0 {
		ctx, canFunc = context.WithTimeout(ctx, time.Millisecond*time.Duration(timeout))
	}

	go func(prom *Promise, ctx context.Context, canFunc context.CancelFunc, path string, value interface{}) {
		if canFunc != nil {
			fmt.Println(canFunc)
			defer canFunc()
		}

		res, err := psgo.Call(ctx, path, value)
		if err != nil {
			fmt.Println("Error: ", err)
			prom.State = REJECT_STATE
			if prom.Catch != nil {
				prom.Catch(err)
			}
		} else {
			prom.State = RESOLVE_STATE
			prom.Answer = res
			if prom.Then != nil {
				prom.Then(res)
			}
		}
	}(prom, ctx, canFunc, path, value)

	return prom
}
