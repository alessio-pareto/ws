package ws

import (
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type Scheduler struct {
	parent *sync.WaitGroup
	child *sync.WaitGroup
	exited bool
	panicChan chan error
}

func (sc *Scheduler) Wait() {
	if sc.child != nil {
		sc.child.Wait()
	}
}

func (sc *Scheduler) Exit() {
	if !sc.exited {
		sc.exited = true
		sc.parent.Done()
	}
}

func (sc *Scheduler) recoverPanic() {
	var panicErr error

	if err := recover(); err != nil {
		switch err := err.(type) {
		case error:
			panicErr = fmt.Errorf("panic: %w\n%s", err, stack())
		default:
			panicErr = fmt.Errorf("panic: %v\n%s", err, stack())
		}

		sc.panicChan <- panicErr
	}

	sc.Exit()
}

func stack() string {
	var out string

	split := strings.Split(string(debug.Stack()), "\n")
	cont := true

	for _, s := range split {
		if strings.HasPrefix(s, "panic(") {
			cont = false
		}

		if cont {
			continue
		}

		out += s + "\n"
	}

	return strings.TrimRight(out, "\n")
}

type SchedulerGoFunc func(sc *Scheduler)

func (sc *Scheduler) Go(f SchedulerGoFunc) {
	if sc.child == nil {
		sc.child = new(sync.WaitGroup)
	}
	sc.child.Add(1)

	childSC := &Scheduler {
		parent: sc.child,
		panicChan: sc.panicChan,
	}
	go childSC.goChildFunc(f)
}

func (sc *Scheduler) GoNB(f SchedulerGoFunc) {
	childSC := &Scheduler {
		parent: sc.child,
		panicChan: sc.panicChan,
	}

	go func() {
		defer time.Sleep(time.Millisecond)
		childSC.goChildFunc(f)
	}()
}

func (sc *Scheduler) goChildFunc(f SchedulerGoFunc) {
	defer sc.recoverPanic()

	f(sc)
}