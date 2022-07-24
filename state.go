package ws

import (
	"time"

	"golang.org/x/sys/windows/svc"
)

type ChangeHandlerFunc func(sm *ServiceManager, c svc.ChangeRequest)

func (sm *ServiceManager) RegisterChangeHandler(cmd svc.Cmd, f ChangeHandlerFunc) {
	sm.s.changeHandlers[cmd] = f
	sm.s.accepts |= acceptedFromCmd(cmd)

	if sm.s.changes != nil {
		sm.s.sendAccepts(sm.s.accepts)
	}
}

func (s *service) tempAccepts(cmd svc.Cmd) svc.Accepted {
	var t svc.Accepted

	for a := range s.changeHandlers {
		if (a == svc.Pause || a == svc.Continue) && (cmd == svc.Pause || cmd == svc.Continue)  {
			continue
		}

		if a == cmd {
			continue
		}

		t |= acceptedFromCmd(a)
	}

	return t
}

func acceptedFromCmd(cmd svc.Cmd) svc.Accepted {
	switch cmd {
	case svc.Continue, svc.Pause:
		return svc.AcceptPauseAndContinue
	case svc.HardwareProfileChange:
		return svc.AcceptHardwareProfileChange
	case svc.NetBindEnable, svc.NetBindDisable, svc.NetBindAdd, svc.NetBindRemove:
		return svc.AcceptNetBindChange
	case svc.ParamChange:
		return svc.AcceptParamChange
	case svc.PowerEvent:
		return svc.AcceptPowerEvent
	case svc.PreShutdown:
		return svc.AcceptPreShutdown
	case svc.SessionChange:
		return svc.AcceptSessionChange
	case svc.Shutdown:
		return svc.AcceptShutdown
	case svc.Stop:
		return svc.AcceptStop
	}

	return 0
}

func (s *service) sendStatus(status svc.Status) {
	if s.changes != nil {
		s.changes <- status
	}
}

func (s *service) sendState(state svc.State) {
	if state == 0 {
		state = s.state
	} else {
		s.state = state
	}

	s.sendStatus(svc.Status{ State: state, Accepts: s.accepts })
}

func (s *service) sendAccepts(accepts svc.Accepted) {
	s.sendStatus(svc.Status{ State: s.state, Accepts: accepts })
}

func (s *service) handleChange(c svc.ChangeRequest) {
	if c.Cmd == svc.Interrogate {
		s.sendStatus(c.CurrentStatus)
		time.Sleep(100 * time.Millisecond)
		s.sendStatus(c.CurrentStatus)

		return
	}

	f, ok := s.changeHandlers[c.Cmd]
	if !ok {
		return
	}

	var before, after *svc.Status

	switch c.Cmd {
	case svc.Pause:
		before = &svc.Status{State: svc.PausePending, Accepts: s.tempAccepts(c.Cmd)}
		after = &svc.Status{State: svc.Paused, Accepts: s.accepts}
	case svc.Continue:
		before = &svc.Status{State: svc.ContinuePending, Accepts: s.tempAccepts(c.Cmd)}
		after = &svc.Status{State: svc.Running, Accepts: s.accepts}
	case svc.Stop, svc.Shutdown, svc.PreShutdown:
		before = &svc.Status{State: svc.StopPending, Accepts: s.tempAccepts(c.Cmd)}
	}

	if before != nil {
		s.sendStatus(*before)
	}
	if f != nil {
		f(s.sm, c)
	}
	if after != nil {
		s.sendStatus(*after)
	}
}