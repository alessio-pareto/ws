package ws

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/mgr"
)

type Option struct {
	typ string
	f   func(s *mgr.Service) error
}

func InstallService(name string, exepath string, cfg mgr.Config, options ...Option) error {
	m, err := managerConnect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.CreateService(name, exepath, cfg)
	if err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	defer s.Close()

	for _, opt := range options {
		err = opt.f(s)
		if err != nil {
			return fmt.Errorf("config option \"%s\": %w", opt.typ, err)
		}
	}

	return nil
}

func RemoveService(name string) error {
	s, err := ConnectToService(name)
	if err != nil {
		return err
	}
	defer s.Close()

	err = s.Delete()
	if err != nil {
		return fmt.Errorf("service delete: %w", err)
	}

	return nil
}

func ArgsOption(args ...string) Option {
	return Option{
		typ: "autostart args",
		f: func(s *mgr.Service) error {
			cfg, err := s.Config()
			if err != nil {
				return err
			}

			for _, a := range args {
				cfg.BinaryPathName += " " + syscall.EscapeArg(a)
			}

			return s.UpdateConfig(cfg)
		},
	}
}

func PreShutdownOption(d time.Duration) Option {
	return Option{
		typ: "preshutdown",
		f: func(s *mgr.Service) error {
			time := d.Milliseconds()
			return windows.ChangeServiceConfig2(
				s.Handle,
				windows.SERVICE_CONFIG_PRESHUTDOWN_INFO,
				(*byte)(unsafe.Pointer(&time)),
			)
		},
	}
}

func FailureActionsOption(failureActions windows.SERVICE_FAILURE_ACTIONS) Option {
	return Option{
		typ: "failure actions",
		f: func(s *mgr.Service) error {
			return windows.ChangeServiceConfig2(
				s.Handle,
				windows.SERVICE_CONFIG_FAILURE_ACTIONS,
				(*byte)(unsafe.Pointer(&failureActions)),
			)
		},
	}
}

func DelayedAutostartOption() Option {
	return Option{
		typ: "delayed autostart",
		f: func(s *mgr.Service) error {
			return windows.ChangeServiceConfig2(
				s.Handle,
				windows.SERVICE_CONFIG_DELAYED_AUTO_START_INFO,
				(*byte)(unsafe.Pointer(&windows.SERVICE_DELAYED_AUTO_START_INFO{ IsDelayedAutoStartUp: 1 })),
			)
		},
	}
}
