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

var idCnt = 0
var subs = map[int]*psgo.Subscriber{}

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
	ob.Set("pub", pub)
	ob.Set("numSubscribers", numSubscribers)
	ob.Set("call", call)
	ob.Set("subscribeFuncs", subscribeFuncs)
	ob.Set("answer", answer)
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

func pub(to string, dat interface{}, opts ...*MsgOpts) {
	msg := &Msg{Object: js.Global.Get("Object").New()}
	msg.To = to
	msg.Dat = dat
	publish(msg, opts...)
}

func numSubscribers(path string) int {
	return psgo.NumSubscribers(path)
}

func call(path string, value interface{}, timeout int64) *js.Object {
	promise := js.Global.Get("Promise").New(func(res, rej func(interface{})) {
		ctx := context.Background()
		var canFunc context.CancelFunc
		if timeout > 0 {
			ctx, canFunc = context.WithTimeout(ctx, time.Millisecond*time.Duration(timeout))
		}

		go func() {
			if canFunc != nil {
				defer canFunc()
			}

			response, err := psgo.Call(ctx, path, value)
			if err != nil {
				if errData, ok := err.(psgo.ErrWithDataInterface); ok {
					rej(errData.Data())
				} else {
					rej(err.Error())
				}
			} else {
				res(response)
			}
		}()
	})
	return promise
}

// subscribeFuncs subscribes using a map with paths as keys and function to execute for each as values. Returns subscription id
func subscribeFuncs(handlers map[string]func(m *Msg)) int {
	id := newSubscriber(func(msg *Msg) {
		handlers[msg.To](msg)
	})

	paths := []string{}
	for k := range handlers {
		paths = append(paths, k)
	}

	subscribe(id, paths...)
	return id
}

func answer(msg *Msg, res, err interface{}) {
	m := &psgo.Msg{To: msg.To, Res: msg.Res, Dat: msg.Dat}
	if err != nil {
		m.Answer(res, errors.New(fmt.Sprintf("%v", err)))
	}
	m.Answer(res, nil)
	return
}
