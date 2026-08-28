//go:build windows

package util

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// serviceStopWaitInterval / serviceStopWaitTimeout 재시작 시 중지 완료를 기다리는 주기와 상한.
const (
	serviceStopWaitInterval = 200 * time.Millisecond
	serviceStopWaitTimeout  = 30 * time.Second
)

// openSCM 필요한 권한만 요청해서 서비스 제어 관리자에 접속한다.
// mgr.Connect()는 SC_MANAGER_ALL_ACCESS를 요구해 관리자 권한이 아니면 조회조차 실패하므로 쓰지 않는다.
func openSCM(access uint32) (*mgr.Mgr, error) {
	handle, err := windows.OpenSCManager(nil, nil, access)
	if err != nil {
		return nil, err
	}
	return &mgr.Mgr{Handle: handle}, nil
}

// openService 지정한 권한으로 서비스 핸들을 연다.
func openService(m *mgr.Mgr, name string, access uint32) (*mgr.Service, error) {
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	handle, err := windows.OpenService(m.Handle, namePtr, access)
	if err != nil {
		return nil, err
	}
	return &mgr.Service{Name: name, Handle: handle}, nil
}

// serviceIsActive Windows 서비스 상태를 systemctl is-active 와 같은 문자열로 변환해 반환한다.
// 서비스가 없거나 조회에 실패하면 "inactive"로 처리한다(리눅스판 동작 유지).
func serviceIsActive(target string) string {
	scm, err := openSCM(windows.SC_MANAGER_CONNECT)
	if err != nil {
		return "inactive"
	}
	defer scm.Disconnect()

	service, err := openService(scm, normalizeServiceName(target), windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return "inactive"
	}
	defer service.Close()

	status, err := service.Query()
	if err != nil {
		return "inactive"
	}

	switch status.State {
	case svc.Running:
		return "active"
	case svc.StartPending, svc.ContinuePending:
		return "activating"
	case svc.StopPending, svc.PausePending:
		return "deactivating"
	default:
		return "inactive"
	}
}

// loadServiceUnits 등록된 Windows 서비스 이름 set을 반환한다. (systemctl list-units 대응)
// 서비스 이름은 대소문자를 구분하지 않으므로 소문자 키로 저장한다.
func loadServiceUnits() map[string]bool {
	units := make(map[string]bool)

	scm, err := openSCM(windows.SC_MANAGER_CONNECT | windows.SC_MANAGER_ENUMERATE_SERVICE)
	if err != nil {
		log.Error(fmt.Errorf("서비스 제어 관리자 접속 실패: %v", err))
		return units
	}
	defer scm.Disconnect()

	names, err := scm.ListServices()
	if err != nil {
		log.Error(fmt.Errorf("Windows 서비스 목록 조회 실패: %v", err))
		return units
	}

	for _, name := range names {
		units[strings.ToLower(name)] = true
	}
	return units
}

// normalizeServiceName Windows 서비스 이름으로 정규화한다.
// 설정에 리눅스 유닛 이름이 그대로 들어와도 동작하도록 .service 접미사만 떼어낸다.
// (서비스 이름은 대소문자를 구분하지 않으므로 원본 표기는 유지한다)
func normalizeServiceName(name string) string {
	name = strings.TrimSpace(name)
	if strings.HasSuffix(strings.ToLower(name), ".service") {
		name = name[:len(name)-len(".service")]
	}
	return name
}

// controlService Windows 서비스를 start/stop/restart 한다.
// restart 는 SCM에 대응 명령이 없으므로 stop 후 중지 완료를 기다렸다가 start 한다.
func controlService(action, serviceName string) error {
	switch strings.ToLower(action) {
	case "start":
		return startWindowsService(serviceName)
	case "stop":
		return stopWindowsService(serviceName)
	case "restart":
		if err := stopWindowsService(serviceName); err != nil {
			// 이미 멈춰 있는 서비스의 restart 는 정상 흐름이므로 로그만 남기고 진행한다.
			log.Warn(fmt.Sprintf("restart 중 stop 실패(무시하고 start 진행): %s: %v", serviceName, err))
		}
		return startWindowsService(serviceName)
	default:
		return fmt.Errorf("unsupported service action: %s", action)
	}
}

func startWindowsService(serviceName string) error {
	scm, err := openSCM(windows.SC_MANAGER_CONNECT)
	if err != nil {
		return err
	}
	defer scm.Disconnect()

	service, err := openService(scm, normalizeServiceName(serviceName), windows.SERVICE_START|windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return err
	}
	defer service.Close()

	return service.Start()
}

func stopWindowsService(serviceName string) error {
	scm, err := openSCM(windows.SC_MANAGER_CONNECT)
	if err != nil {
		return err
	}
	defer scm.Disconnect()

	service, err := openService(scm, normalizeServiceName(serviceName), windows.SERVICE_STOP|windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return err
	}
	defer service.Close()

	status, err := service.Control(svc.Stop)
	if err != nil {
		return err
	}

	// 중지 완료까지 대기 (다음 start 가 ERROR_SERVICE_ALREADY_RUNNING 으로 실패하지 않도록)
	deadline := time.Now().Add(serviceStopWaitTimeout)
	for status.State != svc.Stopped {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for %s to stop", serviceName)
		}
		time.Sleep(serviceStopWaitInterval)
		if status, err = service.Query(); err != nil {
			return err
		}
	}
	return nil
}

// runShell cmd.exe 경유로 명령을 실행한다. (리눅스판 bash -c 대응)
func runShell(command string) error {
	return exec.Command("cmd", "/C", command).Run()
}

/**
 * RestartServer
 * 서버 재시작
 *
 * @return error
 */
func RestartServer() error {
	cmd := exec.Command("shutdown", "/r", "/t", "0")
	err := cmd.Run()
	if err != nil {
		log.Error(fmt.Errorf("restarting server: %v", err))
		return err
	}
	return nil
}

/**
 * ShutdownServer
 * 서버 종료
 *
 * @return error
 */
func ShutdownServer() error {
	cmd := exec.Command("shutdown", "/s", "/t", "0")
	err := cmd.Run()
	if err != nil {
		log.Error(fmt.Errorf("shutting down server: %v", err))
		return err
	}
	return nil
}

// MiddleserverRestart 미들서버(KVM 게스트) 재시작.
// 윈도우에는 libvirt(virsh)가 없으므로 지원하지 않는다.
func MiddleserverRestart() error {
	err := fmt.Errorf("middleserver restart is not supported on windows (requires virsh)")
	log.Error(err)
	return err
}
