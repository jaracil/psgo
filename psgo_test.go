package psgo

import (
	"context"
	"testing"
	"time"
)

func TestSubscribeUnsubscribe(t *testing.T) {
	su := NewSubscriber(func(msg *Msg) {})
	defer su.UnsubscribeAll()
	su.Subscribe("cartas.sota", "cartas.caballo", "cartas.rey")
	if NumSubscribers("cartas.sota") != 1 {
		t.Errorf("numsubscribers must be 1")
	}
	if NumSubscribers("cartas.jota") != 0 {
		t.Errorf("numsubscribers must be 0")
	}
	if su.NumSubscriptions() != 3 {
		t.Errorf("subscriptions must be 3")
	}
	su.Unsubscribe("cartas.rey")
	if su.NumSubscriptions() != 2 {
		t.Errorf("subscriptions must be 2")
	}
	if NumSubscribers("cartas.rey") != 0 {
		t.Errorf("numsubscribers must be 1")
	}
	su.UnsubscribeAll()
	if su.NumSubscriptions() != 0 {
		t.Errorf("subscriptions must be 0")
	}
}
func TestPublishPersistent(t *testing.T) {
	ch := make(chan *Msg, 1)
	f := func(msg *Msg) {
		ch <- msg
	}
	su := NewSubscriber(f)
	defer su.UnsubscribeAll()
	Publish(&Msg{To: "cartas.sota", Dat: 1}, &MsgOpts{Persist: true})
	su.Subscribe("cartas.sota")
	msg := <-ch
	if msg.To != "cartas.sota" {
		t.Errorf("Target error, must be cartas.sota")
	}
	if msg.Dat.(int) != 1 {
		t.Errorf("Data, must be 1")
	}
	if !msg.Old {
		t.Errorf("Old flag error, must true")
	}
	Publish(&Msg{To: "cartas.sota", Dat: 2}, &MsgOpts{Persist: true})
	msg = <-ch
	if msg.To != "cartas.sota" {
		t.Errorf("Target error, must be cartas.sota")
	}
	if msg.Dat.(int) != 2 {
		t.Errorf("Data, must be 2")
	}
	if msg.Old {
		t.Errorf("Old flag error, must false")
	}
}

func TestPublishRecursive(t *testing.T) {
	ch1 := make(chan *Msg, 1)
	f1 := func(msg *Msg) {
		ch1 <- msg
	}
	su1 := NewSubscriber(f1)
	defer su1.UnsubscribeAll()

	ch2 := make(chan *Msg, 1)
	f2 := func(msg *Msg) {
		ch2 <- msg
	}
	su2 := NewSubscriber(f2)
	defer su2.UnsubscribeAll()

	su1.Subscribe("cartas.caballo")
	su2.Subscribe("cartas")

	Publish(&Msg{To: "cartas.caballo", Dat: 1})
	msg := <-ch1
	if msg.To != "cartas.caballo" {
		t.Errorf("Target error, must be cartas.caballo")
	}
	if msg.Dat.(int) != 1 {
		t.Errorf("Data, must be 1")
	}
	if msg.Old {
		t.Errorf("Old flag error, must false")
	}

	msg = <-ch2
	if msg.To != "cartas.caballo" {
		t.Errorf("Target error, must be cartas.caballo")
	}
	if msg.Dat.(int) != 1 {
		t.Errorf("Data, must be 1")
	}
	if msg.Old {
		t.Errorf("Old flag error, must false")
	}
	Publish(&Msg{To: "cartas.rey", Dat: 1})
	msg = <-ch2
	if msg.To != "cartas.rey" {
		t.Errorf("Target error, must be cartas.rey")
	}
	if msg.Dat.(int) != 1 {
		t.Errorf("Data, must be 1")
	}
	if msg.Old {
		t.Errorf("Old flag error, must false")
	}
	if len(ch1) > 0 {
		t.Errorf("ch1 must be empty")
	}

	Publish(&Msg{To: "cartas.caballo", Dat: 2}, &MsgOpts{NoPropagate: true})
	msg = <-ch1
	if msg.To != "cartas.caballo" {
		t.Errorf("Target error, must be cartas.caballo")
	}
	if msg.Dat.(int) != 2 {
		t.Errorf("Data, must be 2")
	}
	if msg.Old {
		t.Errorf("Old flag error, must false")
	}
	if len(ch2) > 0 {
		t.Errorf("ch2 must be empty")
	}
}

func TestCall(t *testing.T) {
	f1 := func(msg *Msg) {
		msg.Answer(msg.Dat)
	}
	su1 := NewSubscriber(f1)
	su1.Subscribe("root.f1")
	defer su1.UnsubscribeAll()
	res, err := Call(context.Background(), "root.f1", "Hello")
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if res.(string) != "Hello" {
		t.Errorf("res must be (Hello)")
	}
}

func TestCallTimeout(t *testing.T) {
	f1 := func(msg *Msg) {
		time.Sleep(time.Millisecond * 20)
		msg.Answer(msg.Dat)
	}
	su1 := NewSubscriber(f1)
	su1.Subscribe("root.f1")
	defer su1.UnsubscribeAll()
	ctx, canFun := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer canFun()
	res, err := Call(ctx, "root.f1", "Hello")
	if err == nil {
		t.Errorf("Expected context done")
	}

	if res != nil {
		t.Errorf("Expected nil result")
	}
}
