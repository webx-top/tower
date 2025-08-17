package rundelay

import (
	"time"
)

func New[T any](delay time.Duration, f func(T) error) *RunDelay[T] {
	d := &RunDelay[T]{
		chNotify: make(chan struct{}),
		chExec:   make(chan struct{}, 1),
		chDone:   make(chan error, 1),
	}
	d.Init(delay, f)
	return d
}

type RunDelay[T any] struct {
	chNotify   chan struct{}
	chExec     chan struct{}
	chDone     chan error
	exec       func(T) error
	delay      time.Duration
	recvNotify func(T, time.Time) // for debug only
}

func (d *RunDelay[T]) Init(delay time.Duration, f func(T) error) {
	d.exec = f
	d.delay = delay
}

func (d *RunDelay[T]) run(v T) {
	d.chDone <- d.exec(v)
	<-d.chExec
}

func (d *RunDelay[T]) delayRun(v T) {
	if d.delay <= 0 {
		d.run(v)
		return
	}
	t := time.NewTicker(d.delay)
	end := time.Now().Add(d.delay)
	defer t.Stop()
	for {
		select {
		case _, ok := <-d.chNotify:
			if !ok {
				return
			}
			t.Reset(d.delay)
			end = time.Now().Add(d.delay)
			if d.recvNotify != nil {
				d.recvNotify(v, end)
			}
		case tm := <-t.C:
			if !tm.Before(end) { // tm >= end
				d.run(v)
				return
			}

			t.Reset(end.Sub(tm))
		}
	}
}

func (d *RunDelay[T]) Run(v T) bool {
	for i, j := 0, len(d.chDone); i < j; i++ {
		<-d.chDone
	}
	select {
	case d.chNotify <- struct{}{}:
	case d.chExec <- struct{}{}:
		d.delayRun(v)
		return true
	default:
	}
	return false
}

func (d *RunDelay[T]) Done() error {
	err := <-d.chDone
	d.chDone <- nil
	return err
}

func (d *RunDelay[T]) Close() error {
	<-d.chDone
	close(d.chExec)
	close(d.chNotify)
	close(d.chDone)
	return nil
}
