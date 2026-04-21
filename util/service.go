package util

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

func ResolveServiceTargets(targetType string) ([]string, error) {
	target := strings.TrimSpace(strings.ToLower(targetType))

	switch target {
	case "back":
		return collectServiceNames(configs.SC.Setting.BackendServiceName), nil
	case "analyze", "main":
		// "main"은 레거시 호환 alias — analyze 서비스와 동일하게 동작
		return splitServiceNames(configs.SC.Setting.AnalyzeServiceName), nil
	case "downloader":
		return splitServiceNames(configs.SC.Setting.DownloaderServiceName), nil
	case "mediaserver":
		return splitServiceNames(configs.SC.Setting.MediaStreamingServiceName), nil
	case "middleserver":
		return []string{"middleserver"}, nil
	case "":
		return nil, fmt.Errorf("service target is empty")
	default:
		return splitServiceNames(targetType), nil
	}
}

// filterExistingServices 서비스 목록에서 실제 존재하는 서비스만 필터링한다.
// middleserver는 검증 없이 통과, docker:*는 compose 파일 존재 여부, 그 외는 systemctl로 확인한다.
func filterExistingServices(targets []string) (existing []string, missing []string) {
	// systemd 서비스 목록을 한 번만 조회
	var systemdUnits map[string]bool
	needSystemd := false
	for _, t := range targets {
		if t != "middleserver" && !strings.Contains(t, "docker") {
			needSystemd = true
			break
		}
	}
	if needSystemd {
		systemdUnits = loadSystemdUnits()
	}

	for _, t := range targets {
		switch {
		case t == "middleserver":
			// virsh 환경 특수성: 검증 건너뜀
			existing = append(existing, t)
		case strings.Contains(t, "docker"):
			// compose 파일 존재 여부 확인
			composePath := fmt.Sprintf("%s/backend/docker-compose.yml", configs.SC.Setting.RootPath)
			if _, err := exec.LookPath("docker"); err == nil {
				if _, err := os.Stat(composePath); err == nil {
					existing = append(existing, t)
					continue
				}
			}
			missing = append(missing, t)
		default:
			// systemd 서비스 확인
			if systemdUnits[t] {
				existing = append(existing, t)
			} else {
				missing = append(missing, t)
			}
		}
	}
	return
}

// loadSystemdUnits systemctl list-units를 한 번 호출해서 로드된 서비스 이름 set을 반환한다.
func loadSystemdUnits() map[string]bool {
	units := make(map[string]bool)
	cmd := exec.Command("bash", "-c", "systemctl list-units --type=service --all --no-legend --no-pager")
	out, err := cmd.Output()
	if err != nil {
		log.Error(fmt.Errorf("systemctl list-units 조회 실패: %v", err))
		return units
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			// 유닛 이름에서 .service 접미사 제거하여 양쪽 모두 매칭
			name := fields[0]
			units[name] = true
			units[strings.TrimSuffix(name, ".service")] = true
		}
	}
	return units
}

func splitServiceNames(serviceNames string) []string {
	names := make([]string, 0)
	for _, serviceName := range strings.Split(serviceNames, ",") {
		serviceName = strings.TrimSpace(serviceName)
		if serviceName == "" {
			continue
		}
		names = append(names, serviceName)
	}
	return names
}

func collectServiceNames(serviceGroups ...string) []string {
	names := make([]string, 0)
	for _, group := range serviceGroups {
		names = append(names, splitServiceNames(group)...)
	}
	return names
}

// getDockerContainerStatus docker inspect로 컨테이너 상태를 확인한다.
// "running"이면 "active", 그 외면 "inactive"를 반환한다.
// 컨테이너가 존재하지 않으면 error를 반환한다.
func getDockerContainerStatus(containerName string) (string, error) {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("docker inspect -f {{.State.Status}} %s", containerName))
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	status := strings.TrimSpace(string(out))
	if status == "running" {
		return "active", nil
	}
	return "inactive", nil
}

// appendRouteServiceStatus OSRM + Nominatim 컨테이너 상태를 "omeye3.route.service"로 집계해
// serviceInfoList에 append한다. 존재하는 컨테이너가 하나도 없으면 항목을 추가하지 않는다.
func appendRouteServiceStatus(serviceInfoList *[]map[string]interface{}) {
	routeContainers := make([]string, 0)
	for _, c := range splitServiceNames(configs.SC.Setting.OSRMContainerName) {
		if c != "" {
			routeContainers = append(routeContainers, c)
		}
	}
	if n := configs.SC.Setting.NominatimContainerName; n != "" {
		routeContainers = append(routeContainers, n)
	}
	if len(routeContainers) == 0 {
		return
	}

	aggregated := "active"
	anyFound := false
	for _, cname := range routeContainers {
		status, err := getDockerContainerStatus(cname)
		if err != nil {
			continue
		}
		anyFound = true
		if status != "active" {
			aggregated = "inactive"
		}
	}
	if anyFound {
		*serviceInfoList = append(*serviceInfoList, map[string]interface{}{
			"serviceName":   "omeye3.route.service",
			"serviceStatus": aggregated,
		})
	}
}

// appendRedisServiceStatus Redis 컨테이너 상태(docker inspect + PING)를 serviceInfoList에 append.
func appendRedisServiceStatus(serviceInfoList *[]map[string]interface{}) {
	name := configs.Redis.RedisContainerName
	if name == "" {
		return
	}
	status, err := getDockerContainerStatus(name)
	if err != nil {
		return
	}
	if status == "active" && !checkRedisPing() {
		status = "inactive"
	}
	*serviceInfoList = append(*serviceInfoList, map[string]interface{}{
		"serviceName":   "redis",
		"serviceStatus": status,
	})
}

// checkRedisPing Redis에 TCP 연결 후 AUTH + PING을 보내 PONG 응답 여부를 확인한다.
func checkRedisPing() bool {
	addr := fmt.Sprintf("%s:%d", configs.Redis.RedisHost, configs.Redis.RedisPort)
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 128)

	// 비밀번호가 설정되어 있으면 AUTH 먼저 전송
	if pw := configs.Redis.Password; pw != "" {
		authCmd := fmt.Sprintf("*2\r\n$4\r\nAUTH\r\n$%d\r\n%s\r\n", len(pw), pw)
		if _, err = conn.Write([]byte(authCmd)); err != nil {
			return false
		}
		n, err := conn.Read(buf)
		if err != nil || !strings.Contains(string(buf[:n]), "+OK") {
			return false
		}
	}

	// PING 전송
	if _, err = conn.Write([]byte("*1\r\n$4\r\nPING\r\n")); err != nil {
		return false
	}

	n, err := conn.Read(buf)
	if err != nil {
		return false
	}
	return strings.Contains(string(buf[:n]), "+PONG")
}

/**
 * GetServiceStatus
 * 서비스 상태 정보 조회
 *
 * @return map[string]interface{}
 * @return error
 *
 * @autor: Han Seong San
 * @version: 1.0.0
 * @since: 2024.06.20
 */
func GetServiceStatus() ([]map[string]interface{}, error) {
	serviceInfoList := make([]map[string]interface{}, 0)

	for _, target := range activeServices() {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}

		if strings.Contains(target, "docker") {
			// Docker 컨테이너 상태 확인
			containerArr := strings.Split(target, ":")
			if len(containerArr) < 2 {
				continue
			}
			status, err := getDockerContainerStatus(containerArr[1])
			if err != nil {
				log.Error(fmt.Errorf("fetching %s container info: %v", containerArr[1], err))
				continue
			}
			serviceInfoList = append(serviceInfoList, map[string]interface{}{
				"serviceName":   containerArr[1],
				"serviceStatus": status,
			})
		} else {
			// systemd 서비스 상태 확인
			cmd := exec.Command("bash", "-c", fmt.Sprintf("systemctl is-active %s", target))
			out, _ := cmd.Output()
			active := strings.TrimSpace(string(out))
			// is-active 결과: active, inactive, failed, activating, deactivating 등
			if active == "" {
				active = "inactive"
			}
			serviceInfoList = append(serviceInfoList, map[string]interface{}{
				"serviceName":   target,
				"serviceStatus": active,
			})
		}
	}

	// serverType 별 docker 서비스 모니터링 (config의 mainDockerServices / analyzeDockerServices)
	for _, item := range activeDockerServices() {
		switch strings.ToLower(strings.TrimSpace(item)) {
		case "route":
			appendRouteServiceStatus(&serviceInfoList)
		case "redis":
			appendRedisServiceStatus(&serviceInfoList)
		default:
			log.Warn("unknown docker service keyword: " + item)
		}
	}

	return serviceInfoList, nil
}

/**
 * RestartService
 * 서비스 재시작
 *
 * @param serviceName string
 * @return error
 *
 * @autor: Han Seong San
 * @version: 1.0.0
 * @since: 2024.06.20
 */
func RestartService(serviceName string) error {
	var command string
	if strings.Contains(serviceName, "docker") {
		// TODO: docker compose 관련 재시작은 서비스 파일로 등록
		command = fmt.Sprintf("docker compose -f %s restart", fmt.Sprintf("%s/backend/docker-compose.yml", configs.SC.Setting.RootPath))
	} else {
		command = fmt.Sprintf("systemctl restart %s", serviceName)
	}
	cmd := exec.Command("bash", "-c", command)
	log.Info(fmt.Sprintf("Restarting Service: %s", command))
	err := cmd.Run()
	if err != nil {
		log.Error(fmt.Errorf("restarting %s: %v", serviceName, err))
		return err
	}
	return nil
}

/**
 * StopService
 * 서비스 중지
 *
 * @param serviceName string
 * @return error
 *
 * @autor: Han Seong San
 * @version: 1.0.0
 * @since: 2024.06.20
 */
func StopService(serviceName string) error {
	var command string
	if strings.Contains(serviceName, "docker") {
		command = fmt.Sprintf("docker compose -f %s stop", fmt.Sprintf("%s/conf.d/docker-compose.yml", configs.SC.Setting.RootPath))
	} else {
		command = fmt.Sprintf("systemctl stop %s", serviceName)
	}
	cmd := exec.Command("bash", "-c", command)
	log.Info(fmt.Sprintf("Stopping Service: %s", command))
	err := cmd.Run()
	if err != nil {
		log.Error(fmt.Errorf("stopping %s: %v", serviceName, err))
		return err
	}
	return nil
}

/**
 * StartService
 * 서비스 시작
 *
 * @param serviceName string
 * @return error
 *
 * @autor: Han Seong San
 * @version: 1.0.0
 * @since: 2024.06.20
 */
func StartService(serviceName string) error {
	var command string
	if strings.Contains(serviceName, "docker") {
		command = fmt.Sprintf("docker compose -f %s start", fmt.Sprintf("%s/conf.d/docker-compose.yml", configs.SC.Setting.RootPath))
	} else {
		command = fmt.Sprintf("systemctl start %s", serviceName)
	}
	cmd := exec.Command("bash", "-c", command)
	log.Info(fmt.Sprintf("Starting Service: %s", command))
	err := cmd.Run()
	if err != nil {
		log.Error(fmt.Errorf("starting %s: %v", serviceName, err))
		return err
	}
	return nil
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
