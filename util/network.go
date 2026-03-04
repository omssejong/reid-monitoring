package util

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type NetworkUsage struct {
	bytesSent uint64
	bytesRecv uint64
}

// GetNetworkInfo returns interface IP/netmask and default gateway without shell commands.
func GetNetworkInfo(interfaceName string) (map[string]interface{}, error) {
	networkDict := make(map[string]interface{})

	ifc, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, err
	}

	addrs, err := ifc.Addrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ipv4 := ipNet.IP.To4()
		if ipv4 == nil {
			continue
		}
		networkDict["ip"] = ipv4.String()
		networkDict["netmask"] = net.IP(ipNet.Mask).String()
		break
	}

	gateway, err := getDefaultGateway(interfaceName)
	if err != nil {
		return nil, err
	}
	if gateway != "" {
		networkDict["gateway"] = gateway
	}

	return networkDict, nil
}

// GetNetworkBandwidth reads link speed from sysfs (Mbps -> Mb/s string).
func GetNetworkBandwidth(interfaceName string) (string, error) {
	speedPath := filepath.Join("/sys/class/net", interfaceName, "speed")
	content, err := os.ReadFile(speedPath)
	if err != nil {
		return "unknown", nil
	}

	speedText := strings.TrimSpace(string(content))
	if speedText == "" || speedText == "-1" {
		return "unknown", nil
	}

	speedMbps, err := strconv.Atoi(speedText)
	if err != nil || speedMbps <= 0 {
		return "unknown", nil
	}

	return fmt.Sprintf("%dMb/s", speedMbps), nil
}

// GetNetworkUsage reads sent/recv byte counters from /proc/net/dev.
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

func getDefaultGateway(interfaceName string) (string, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		if fields[0] != interfaceName {
			continue
		}
		if fields[1] != "00000000" {
			continue
		}
		return parseLittleEndianHexIPv4(fields[2])
	}

	return "", scanner.Err()
}

func parseLittleEndianHexIPv4(hexStr string) (string, error) {
	if len(hexStr) != 8 {
		return "", fmt.Errorf("invalid hex length: %s", hexStr)
	}

	raw, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}
	if len(raw) != 4 {
		return "", fmt.Errorf("invalid ipv4 hex: %s", hexStr)
	}

	ip := net.IPv4(raw[3], raw[2], raw[1], raw[0])
	return ip.String(), nil
}
