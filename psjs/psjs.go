package psjs

import (
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
	ob.Set("numSubscribers", numSubscribers)

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
