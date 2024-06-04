package ws

import (
	"time"

	"golang.org/x/sys/windows/svc"
)

type ChangeHandlerFunc func(s *Service, c svc.ChangeRequest)

func (s *Service) RegisterChangeHandler(cmd svc.Cmd, f ChangeHandlerFunc) {
	s.changeHandlers[cmd] = f
	s.accepts |= acceptedFromCmd(cmd)

	if s.changes != nil {
		s.SendAccepts(s.accepts)
	}
}

func (s *Service) tempAccepts(cmd svc.Cmd) svc.Accepted {
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

func (s *Service) SendStatus(status svc.Status) {
	s.changes <- status
}

func (s *Service) SendState(state svc.State) {
	if state == 0 {
		state = s.state
	} else {
		s.state = state
	}

	s.SendStatus(svc.Status{ State: state, Accepts: s.accepts })
}

func (s *Service) SendAccepts(accepts svc.Accepted) {
	s.SendStatus(svc.Status{ State: s.state, Accepts: accepts })
}

func (s *Service) handleChange(c svc.ChangeRequest) {
	if c.Cmd == svc.Interrogate {
		s.SendStatus(c.CurrentStatus)
		time.Sleep(100 * time.Millisecond)
		s.SendStatus(c.CurrentStatus)

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
		s.SendStatus(*before)
	}
	if f != nil {
		f(s, c)
	}
	if after != nil {
		s.SendStatus(*after)
	}
}
