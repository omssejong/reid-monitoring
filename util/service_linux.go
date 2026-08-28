//go:build linux

package util

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// camelCasePattern 서비스 이름의 camelCase 경계. systemd 유닛 관례에 맞춰 kebab-case로 바꾼다.
var camelCasePattern = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// serviceIsActive systemctl is-active를 timeout과 함께 실행해 상태 문자열을 반환한다.
// 타임아웃/실패 시 빈 출력 → "inactive"로 처리(기존 동작 유지).
func serviceIsActive(target string) string {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()
	out, _ := exec.CommandContext(ctx, "systemctl", "is-active", target).Output()
	active := strings.TrimSpace(string(out))
	if active == "" {
		return "inactive"
	}
	return active
}

// loadServiceUnits systemctl list-units를 한 번 호출해서 로드된 서비스 이름 set을 반환한다.
func loadServiceUnits() map[string]bool {
	units := make(map[string]bool)
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "systemctl", "list-units", "--type=service", "--all", "--no-legend", "--no-pager")
	out, err := cmd.Output()
	if err != nil {
		log.Error(fmt.Errorf("systemctl list-units 조회 실패: %v", err))
		return units
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			// 유닛 이름에서 .service 접미사 제거하여 양쪽 모두 매칭
			name := strings.ToLower(fields[0])
			units[name] = true
			units[strings.TrimSuffix(name, ".service")] = true
		}
	}
	return units
}

// normalizeServiceName systemd 유닛 관례(kebab-case, 소문자)에 맞게 이름을 정규화한다.
func normalizeServiceName(name string) string {
	return strings.ToLower(camelCasePattern.ReplaceAllString(strings.TrimSpace(name), `${1}-${2}`))
}

// controlService systemctl로 서비스를 start/stop/restart 한다.
func controlService(action, serviceName string) error {
	return runShell(fmt.Sprintf("systemctl %s %s", action, serviceName))
}

// runShell bash 경유로 명령을 실행한다.
func runShell(command string) error {
	return exec.Command("bash", "-c", command).Run()
}

/**
 * RestartServer
 * 서버 재시작
 *
 * @return error
 *
 * @autor: Han Seong San
 * @version: 1.0.0
 * @since: 2024.06.20
 */
func RestartServer() error {
	cmd := exec.Command("bash", "-c", "reboot")
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
 *
 * @autor: Han Seong San
 * @version: 1.0.0
 * @since: 2024.06.20
 */
func ShutdownServer() error {
	cmd := exec.Command("bash", "-c", "shutdown now")
	err := cmd.Run()
	if err != nil {
		log.Error(fmt.Errorf("shutting down server: %v", err))
		return err
	}
	return nil
}

func MiddleserverRestart() error {
	// root 권한으로 실행되므로 sudo 불필요
	cmd := exec.Command("bash", "-c", fmt.Sprintf("virsh reboot %s --mode acpi", configs.SC.Setting.MediaStreamingServiceName))
	err := cmd.Run()
	if err != nil {
		log.Error(fmt.Errorf("error rebooting middle server: %v", err))
		return err
	}
	return nil
}
