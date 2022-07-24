package ws

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

type SvcHandlerFunc func(sm *ServiceManager, sc *Scheduler, args ...string)

type SvcOutErrFunc func() (out, err io.Writer, close func())

func (sm *ServiceManager) Start(outErr SvcOutErrFunc, handlerFunc SvcHandlerFunc, args ...string) error {
	inService, err := svc.IsWindowsService()
	if err != nil {
		return fmt.Errorf("failed to determine if we are running in service: %w", err)
	}

	if inService {
		defer func() {
			if err := recover(); err != nil {
				switch err := err.(type) {
				case error:
					sm.panicErr = fmt.Errorf("%s service start panic: %w\n%s", sm.name, err, stack())
				default:
					sm.panicErr = fmt.Errorf("%s service start panic: %v\n%s", sm.name, err, stack())
				}
			}
		}()

		sm.inService = true

		err = sm.changeWD()
		if err != nil {
			return err
		}
		
		var outErrCloseFunc func()
		sm.out, sm.err, outErrCloseFunc = outErr()
		defer outErrCloseFunc()

		sm.s.handlerFunc = handlerFunc
		return sm.run()
	}

	return sm.startService(args...)
}

func (sm *ServiceManager) run() error {
	err := svc.Run(sm.name, sm.s)
	if err != nil {
		return fmt.Errorf("%s service failed: %w", sm.name, err)
	}

	return nil
}

func (sm *ServiceManager) startService(args ...string) error {
	s, err := sm.connectToService()
	if err != nil {
		return err
	}
	defer s.Close()

	err = s.Start(args...)
	if err != nil {
		return fmt.Errorf("could not start service: %v", err)
	}
	return err
}

func (sm *ServiceManager) changeWD() error {
	s, err := sm.connectToService()
	if err != nil {
		return err
	}
	defer s.Close()

	cfg, err := s.Config()
	if err != nil {
		return err
	}

	return os.Chdir(path.Dir(strings.ReplaceAll(cfg.BinaryPathName, "\\", "/")))
}

func (sm *ServiceManager) ControlService(c svc.Cmd, to svc.State) error {
	s, err := sm.connectToService()
	if err != nil {
		return err
	}
	defer s.Close()

	status, err := s.Control(c)
	if err != nil {
		return fmt.Errorf("could not send control=%d: %v", c, err)
	}

	timeout := time.Now().Add(time.Second * 10)
	for status.State != to {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to go to state=%d", to)
		}

		time.Sleep(time.Millisecond * 300)

		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service status: %v", err)
		}
	}

	return nil
}

func (sm *ServiceManager) Stop() error {
	return sm.ControlService(svc.Stop, svc.Stopped)
}

func (sm *ServiceManager) Pause() error {
	return sm.ControlService(svc.Pause, svc.Paused)
}

func (sm *ServiceManager) Continue() error {
	return sm.ControlService(svc.Continue, svc.Running)
}

func (sm *ServiceManager) connectToService() (s *mgr.Service, err error) {
	m, err := mgr.Connect()
	if err != nil {
		return
	}
	defer m.Disconnect()

	s, err = m.OpenService(sm.name)
	if err != nil {
		return nil, fmt.Errorf("could not access service: %v", err)
	}
	
	return
}
