package ws

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func (s *Service) Run(args ...string) error {
	err := svc.Run(s.name, s)
	if err != nil {
		return fmt.Errorf("service failed: %w", err)
	}

	return nil
}

func SendCommand(name string, c svc.Cmd, to svc.State, waitTime time.Duration) error {
	s, err := ConnectToService(name)
	if err != nil {
		return err
	}
	defer s.Close()

	status, err := s.Control(c)
	if err != nil {
		return fmt.Errorf("send control %d: %w", c, err)
	}

	if status.State == to {
		return nil
	}

	return WaitForState(s, to, waitTime)
}

func WaitForState(s *mgr.Service, to svc.State, waitTime time.Duration) error {
	timeout := time.Now().Add(waitTime)
	
	for  {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for state %d", to)
		}

		time.Sleep(time.Millisecond * 50)

		status, err := s.Query()
		if err != nil {
			return fmt.Errorf("service status: %w", err)
		}

		if status.State == to {
			break
		}

		if status.State == svc.Stopped {
			return errors.New("service stopped unexpectedly")
		}
	}

	return nil
}

func Start(name string, waitTime time.Duration, args ...string) error {
	s, err := ConnectToService(name)
	if err != nil {
		return err
	}
	defer s.Close()

	err = s.Start(args...)
	if err != nil {
		return fmt.Errorf("service start: %w", err)
	}

	return WaitForState(s, svc.Running, waitTime)
}

func Stop(name string, waitTime time.Duration) error {
	return SendCommand(name, svc.Stop, svc.Stopped, waitTime)
}

func Pause(name string, waitTime time.Duration) error {
	return SendCommand(name, svc.Pause, svc.Paused, waitTime)
}

func Continue(name string, waitTime time.Duration) error {
	return SendCommand(name, svc.Continue, svc.Running, waitTime)
}

func managerConnect() (*mgr.Mgr, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("service manager connection: %w", err)
	}
	return m, nil
}

func ConnectToService(name string) (*mgr.Service, error) {
	m, err := managerConnect()
	if err != nil {
		return nil, err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return nil, fmt.Errorf("service open: %w", err)
	}
	
	return s, nil
}
