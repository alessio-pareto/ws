package ws

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

type ServiceManager struct {
	name string
	displayName string
	description string
	s *service
	out io.Writer
	err io.Writer
	panicErr error
	inService bool
}

func Service(name, displayName, description string) *ServiceManager {
	sm := &ServiceManager {
		name: name,
		displayName: displayName,
		description: description,
		s: &service{
			changeHandlers: make(map[svc.Cmd]ChangeHandlerFunc),
		},
	}

	sm.s.sm = sm
	return sm
}

func (sm *ServiceManager) Name() string {
	return sm.name
}

func (sm *ServiceManager) DisplayName() string {
	return sm.displayName
}

func (sm *ServiceManager) Description() string {
	return sm.description
}

func (sm *ServiceManager) IsInService() bool {
	return sm.inService
}

func (sm *ServiceManager) PanicErr() error {
	return sm.panicErr
}

func (sm *ServiceManager) Run(out SvcOutErrFunc, handler SvcHandlerFunc) error {
	if len(os.Args) > 1 {
		var err error
		cmd := strings.ToLower(os.Args[1])

		switch cmd {
		case "install":
			err = sm.InstallService(mgr.Config {
				StartType: mgr.StartManual,
				ErrorControl: mgr.ErrorIgnore,
				DisplayName: sm.displayName,
				Description: sm.description,
			})
		case "remove":
			err = sm.RemoveService()
		case "start":
			err = sm.Start(out, handler)
		case "stop":
			err = sm.Stop()
		case "pause":
			err = sm.Pause()
		case "continue":
			err = sm.Continue()
		default:
			cmd = ""
			err = sm.Start(out, handler, os.Args[1:]...)
		}

		if err != nil {
			if cmd == "" {
				return fmt.Errorf("%s - error: %w", sm.Name(), err)
			}
			
			return fmt.Errorf("%s - error with command <%s>: %w", sm.Name(), cmd, err)
		}

		return nil
	}

	return sm.Start(out, handler, os.Args[1:]...)
}

func (sm *ServiceManager) Started() {
	sm.s.sendState(svc.Running)
}

type service struct {
	sm *ServiceManager
	handlerFunc SvcHandlerFunc
	state svc.State
	changes chan<- svc.Status
	changeHandlers map[svc.Cmd]ChangeHandlerFunc
	accepts svc.Accepted
}

func (s *service) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	s.changes = changes
	s.sendState(svc.StartPending)

	defer func() {
		s.changes <- svc.Status{ State: svc.Stopped }
	}()

	s.execute(args, r, changes)
	return
}

func (s *service) execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) {
	panicChan := make(chan error, 10)
	defer close(panicChan)
	go func() {
		for err := range panicChan {
			s.sm.Logln(err)
		}
	}()

	sc := &Scheduler{ panicChan: panicChan }

	sc.GoNB(func(sc *Scheduler) {
		for c := range r {
			s.handleChange(c)
		}
	})

	if s.changeHandlers[svc.Stop] == nil {
		s.sm.RegisterChangeHandler(svc.Stop, func(sm *ServiceManager, c svc.ChangeRequest) {
			panic(fmt.Errorf("%s service received stop signal but no handler was registered", s.sm.name))
		})
	}

	sc.Go(func(sc *Scheduler) {
		s.handlerFunc(s.sm, sc, args...)
	})

	sc.Wait()
}
