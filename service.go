package ws

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
)

type ServiceManager struct {
	name string
	displayName string
	description string
	s *service
	inService bool
}

func Service(name, displayName, description string) (*ServiceManager, error) {
	sm := &ServiceManager {
		name: name,
		displayName: displayName,
		description: description,
		s: &service{
			changeHandlers: make(map[svc.Cmd]ChangeHandlerFunc),
		},
	}

	var err error
	sm.inService, err = svc.IsWindowsService()
	if err != nil {
		return nil, fmt.Errorf("failed to determine if we are running in service: %w", err)
	}

	sm.s.sm = sm
	return sm, nil
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

func (sm *ServiceManager) Run(handler SvcHandlerFunc) error {
	if len(os.Args) > 1 {
		var err error
		cmd := strings.ToLower(os.Args[1])

		switch cmd {
		case "install":
			err = sm.InstallService(sm.DefaultServiceConfig(), time.Second * 10)
		case "remove":
			err = sm.RemoveService()
		case "start":
			err = sm.Start(handler)
		case "stop":
			err = sm.Stop()
		case "pause":
			err = sm.Pause()
		case "continue":
			err = sm.Continue()
		default:
			cmd = ""
			err = sm.Start(handler, os.Args[1:]...)
		}

		if err != nil {
			if cmd == "" {
				return fmt.Errorf("%s - error: %w", sm.Name(), err)
			}
			
			return fmt.Errorf("%s - error with command <%s>: %w", sm.Name(), cmd, err)
		}

		return nil
	}

	return sm.Start(handler)
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

	s.execute(args, r)
	return
}

func (s *service) execute(args []string, r <-chan svc.ChangeRequest) {
	go func() {
		for c := range r {
			s.handleChange(c)
		}
	}()

	s.handlerFunc(s.sm, args...)
}
