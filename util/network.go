package util

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	goutilnet "github.com/shirou/gopsutil/v4/net"
)

type NetworkUsage struct {
	bytesSent uint64
	bytesRecv uint64
}

/**
 * GetNetworkInfo
 * 네트워크 정보를 가져옴
 *
 * @param name string
 * @return map[string]interface{}
 * @return error
 *
 * @autor: Han Seong San
 * @since: 2024.06.20
 * @version: 1.0.0
 */
func GetNetworkInfo(interfaceName string) (map[string]interface{}, error) {

	networkDict := make(map[string]interface{})

	// Get ip address, netmask
	cmd := exec.Command("ifconfig", networkName)

	interfaces, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	networkArr := make([]string, 0)
	items := strings.Split(strings.Split(string(interfaces), "\n")[1], " ")
	for _, item := range items {
		if item == "" {
			continue
		}
		networkArr = append(networkArr, item)
	}

	for i, network := range networkArr {
		switch network {
		case "inet":
			networkDict["ip"] = networkArr[i+1]
		case "netmask":
			networkDict["netmask"] = networkArr[i+1]
		}
	}

	// Get gateway
	cmd = exec.Command("route", "-n")
	routes, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	gatewayArr := make([][]string, 0)
	for _, route := range strings.Split(string(routes), "\n") {
		if strings.Contains(route, interfaceName) {
			targets := make([]string, 0)
			for _, item := range strings.Split(route, " ") {
				if item == "" {
					continue
				}
				targets = append(targets, item)
			}

			if len(targets) > 0 {
				gatewayArr = append(gatewayArr, targets)
			}
		}
	}

	for _, gateway := range gatewayArr {
		if gateway[1] != "0.0.0.0" {
			networkDict["gateway"] = gateway[1]
		}
	}

	return networkDict, nil
}

/**
 * GetNetworkBandwidth
 * 네트워크 대역폭을 가져옴
 *
 * @param interfaceName string
 * @return string
 * @return error
 *
 * @autor: Han Seong San
 * @since: 2024.06.20
 * @version: 1.0.0
 */
func GetNetworkBandwidth(interfaceName string) (string, error) {
	cmd := exec.Command("ethtool", interfaceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Speed:") {
			return strings.TrimSpace(strings.Split(line, ":")[1]), nil
		}
	}

	return "", nil
}

/**
 * GetNetworkTraffic
 * 네트워크 트래픽을 가져옴
 *
 * @param interfaceName string
 * @return map[string]interface{}
 * @return error
 *
 * @autor: Han Seong San
 * @since: 2024.06.20
 * @version: 1.0.0
 */
func GetNetworkTraffic(interfaceName string) (map[string]interface{}, error) {
	trafficDict := make(map[string]interface{})

	netStat, err := goutilnet.IOCounters(true)
	if err != nil {
		return nil, err
	}

	connStat, err := goutilnet.IOCountersWithContext(context.Background(), true)

	for _, stat := range netStat {
		if stat.Name == interfaceName {
			trafficDict["bytes_sent"] = stat.BytesSent
			trafficDict["bytes_recv"] = stat.BytesRecv
			trafficDict["packets_sent"] = stat.PacketsSent
			trafficDict["packets_recv"] = stat.PacketsRecv
			trafficDict["err_in"] = stat.Errin
			trafficDict["err_out"] = stat.Errout
			trafficDict["drop_in"] = stat.Dropin
			trafficDict["drop_out"] = stat.Dropout
		}
	}

	for _, stat := range connStat {
		trafficDict["bytes_sent_2"] = stat.PacketsSent
		trafficDict["bytes_recv_2"] = stat.PacketsRecv
	}

	return trafficDict, nil
}

/**
 * GetNetworkUsage
 * 네트워크 사용량을 가져옴
 *
 * @param interfaceName string
 * @return NetworkUsage
 * @return error
 *
 * @autor: Han Seong San
 * @since: 2024.07.10
 * @version: 1.0.0
 */
func GetNetworkUsage(interfaceName string) (NetworkUsage, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return NetworkUsage{}, err
	}
	defer file.Close()

	usage := NetworkUsage{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, fmt.Sprintf("%s:", interfaceName)) {
			fields := strings.Fields(line)
			recvBytes, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return NetworkUsage{}, err
			}
			sentBytes, err := strconv.ParseUint(fields[9], 10, 64)
			if err != nil {
				return NetworkUsage{}, err
			}
			usage.bytesRecv += recvBytes
			usage.bytesSent += sentBytes
		}
	}

	return usage, scanner.Err()
}
