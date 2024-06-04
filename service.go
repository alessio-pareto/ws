package ws

import (
	"golang.org/x/sys/windows/svc"
)

type HandlerFunc func(s *Service, args ...string)

type Service struct {
	name string
	handlerFunc HandlerFunc
	state svc.State
	changes chan<- svc.Status
	changeHandlers map[svc.Cmd]ChangeHandlerFunc
	accepts svc.Accepted
}

func ServiceHandler(name string, handlerFunc HandlerFunc) *Service {
	return &Service{
		name: name,
		handlerFunc: handlerFunc,
		changeHandlers: make(map[svc.Cmd]ChangeHandlerFunc),
	}
}

func (s *Service) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	s.changes = changes
	s.SendState(svc.StartPending)

	defer func() {
		s.changes <- svc.Status{ State: svc.Stopped }
	}()

	go func() {
		for c := range r {
			s.handleChange(c)
		}
	}()

	s.handlerFunc(s, args...)
	return
}
