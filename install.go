package ws

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/mgr"
)

func exePath() (string, error) {
	prog := os.Args[0]

	p, err := filepath.Abs(prog)
	if err != nil {
		return "", err
	}

	fi, err := os.Stat(p)
	if err == nil {
		if !fi.Mode().IsDir() {
			return p, nil
		}
		err = fmt.Errorf("%s is directory", p)
	}

	if filepath.Ext(p) == "" {
		p += ".exe"
		fi, err := os.Stat(p)
		if err == nil {
			if !fi.Mode().IsDir() {
				return p, nil
			}
			err = fmt.Errorf("%s is directory", p)
			return "", err
		}
	}

	return "", err
}

func (sm *ServiceManager) InstallService(cfg mgr.Config) error {
	exepath, err := exePath()
	if err != nil {
		return err
	}

	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(sm.name)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", sm.name)
	}

	if cfg.BinaryPathName == "" {
		cfg.BinaryPathName = exepath
	}

	s, err = m.CreateService(sm.name, exepath, cfg)
	if err != nil {
		return err
	}
	defer s.Close()

	time := 2000

	return windows.ChangeServiceConfig2(s.Handle, windows.SERVICE_CONFIG_PRESHUTDOWN_INFO, (*byte)(unsafe.Pointer(&time)))
}

func (sm *ServiceManager) RemoveService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(sm.name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", sm.name)
	}
	defer s.Close()

	err = s.Delete()
	if err != nil {
		return err
	}

	return nil
}
