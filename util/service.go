package util

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

func ResolveServiceTargets(targetType string) ([]string, error) {
	target := strings.TrimSpace(strings.ToLower(targetType))

	switch target {
	case "back":
		return splitServiceNames(configs.SC.Setting.BackendServiceName), nil
	case "main":
		return splitServiceNames(configs.SC.Setting.AiServiceName), nil
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

	// 정규표현식으로 특수문자를 확인하는 패턴
	re := regexp.MustCompile(`[^a-zA-Z0-9\s._-]`)

	for _, target := range strings.Split(monitoringTarget, ",") {
		services := make([][]string, 0)

		var command string
		var containerArr []string
		if strings.Contains(target, "docker") {
			containerArr = strings.Split(target, ":")
			command = fmt.Sprintf("docker inspect -f {{.State.Status}} %s", containerArr[1])
		} else {
			command = fmt.Sprintf("systemctl list-units --type=service --all | grep %s", target)
		}
		cmd := exec.Command("bash", "-c", command)
		servicesByte, err := cmd.CombinedOutput()
		if err != nil {
			log.Error(fmt.Errorf("fetching %s service info: %v", target, err))
			continue
		}

		for _, service := range strings.Split(string(servicesByte), "\n") {
			serviceArr := make([]string, 0)
			if service == "" {
				continue
			}
			orgServiceArr := strings.Split(service, " ")
			for _, orgService := range orgServiceArr {
				if orgService == "" {
					continue
				}
				// 정규표현식으로 특수문자 존재 시 append 하지 않는 로직
				if !re.MatchString(orgService) {
					serviceArr = append(serviceArr, orgService)
				}
			}
			services = append(services, serviceArr)
		}

		for _, service := range services {
			serviceInfoDict := make(map[string]interface{})
			var serviceName, loaded, active string
			if len(service) == 1 {
				serviceName = "omeye2_back_service"
				loaded = "loaded"
				if service[0] == "running" {
					active = "active"
				} else {
					active = "inactive"
				}
			} else {
				serviceName = service[0]
				loaded = service[1]
				active = service[2]
			}

			//log.Info(fmt.Sprintf("Service Name: %s, Loaded: %s, Active: %s", serviceName, loaded, active))

			if loaded == "loaded" {
				serviceInfoDict["serviceName"] = serviceName
				serviceInfoDict["serviceStatus"] = active
				serviceInfoList = append(serviceInfoList, serviceInfoDict)
			}

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
		command = fmt.Sprintf("echo %s | sudo -S docker compose -f %s restart", password, fmt.Sprintf("%s/backend/docker-compose.yml", configs.SC.Setting.RootPath))
	} else {
		command = fmt.Sprintf("echo %s | sudo -S systemctl restart %s", password, serviceName)
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
		command = fmt.Sprintf("echo %s | sudo -S docker compose -f %s stop", password, fmt.Sprintf("%s/conf.d/docker-compose.yml", configs.SC.Setting.RootPath))
	} else {
		command = fmt.Sprintf("echo %s | sudo -S systemctl stop %s", password, serviceName)
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
		command = fmt.Sprintf("echo %s | sudo -S docker compose -f %s start", password, fmt.Sprintf("%s/conf.d/docker-compose.yml", configs.SC.Setting.RootPath))
	} else {
		command = fmt.Sprintf("echo %s | sudo -S systemctl start %s", password, serviceName)
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
	cmd := exec.Command("bash", "-c", fmt.Sprintf("echo %s | sudo -S reboot", password))
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
	cmd := exec.Command("bash", "-c", fmt.Sprintf("echo %s | sudo -S shutdown now", password))
	err := cmd.Run()
	if err != nil {
		log.Error(fmt.Errorf("shutting down server: %v", err))
		return err
	}
	return nil
}

func MiddleserverRestart() error {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("echo %s | sudo virsh reboot %s --mode acpi", password, configs.SC.Setting.MediaStreamingServiceName))
	err := cmd.Run()
	if err != nil {
		log.Error(fmt.Errorf("error rebooting middle server: %v", err))
		return err
	}
	return nil
}
