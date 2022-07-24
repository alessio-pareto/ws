package ws

import (
	"fmt"
	"io"
)

func (sm *ServiceManager) Out() io.Writer {
	return sm.out
}

func (sm *ServiceManager) Err() io.Writer {
	return sm.err
}

func (sm *ServiceManager) Println(a ...any) (n int, err error) {
	return fmt.Fprintln(sm.out, a...)
}

func (sm *ServiceManager) Printf(format string, a ...any) (n int, err error) {
	return fmt.Fprintf(sm.out, format, a...)
}

func (sm *ServiceManager) Print(a ...any) (n int, err error) {
	return fmt.Fprint(sm.out, a...)
}

func (sm *ServiceManager) Logln(a ...any) (n int, err error) {
	return fmt.Fprintln(sm.err, a...)
}

func (sm *ServiceManager) Logf(format string, a ...any) (n int, err error) {
	return fmt.Fprintf(sm.err, format, a...)
}

func (sm *ServiceManager) Log(a ...any) (n int, err error) {
	return fmt.Fprint(sm.err, a...)
}

/* func (sm *ServiceManager) Fatalln(a ...any) (n int, err error) {
	return fmt.Fprintln(sm.err, a...)
	// sm.triggerExit()
}

func (sm *ServiceManager) Fatalf(format string, a ...any) (n int, err error) {
	return fmt.Fprintf(sm.err, format, a...)
	// sm.triggerExit()
}

func (sm *ServiceManager) Fatal(a ...any) (n int, err error) {
	return fmt.Fprint(sm.err, a...)
	// sm.triggerExit()
} */