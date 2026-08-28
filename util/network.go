package util

import (
	"fmt"
	"net"
)

// 플랫폼 의존 함수는 network_linux.go / network_windows.go 에서 구현한다.
//   - GetNetworkBandwidth : 인터페이스 링크 속도
//   - GetNetworkUsage     : 인터페이스 누적 송수신 바이트
//   - getDefaultRoute     : 기본 게이트웨이가 붙은 인터페이스 이름과 게이트웨이 IP

type NetworkUsage struct {
	bytesSent uint64
	bytesRecv uint64
}

type NetworkInterfaceInfo struct {
	Name      string `json:"name"`
	IP        string `json:"ip"`
	Netmask   string `json:"netmask"`
	Gateway   string `json:"gateway,omitempty"`
	Bandwidth string `json:"bandwidth"`
	IsDefault bool   `json:"isDefault"`
}

// GetNetworkInterfaces returns every active IPv4-capable interface.
func GetNetworkInterfaces() ([]NetworkInterfaceInfo, error) {
	defaultIface, defaultGateway, err := getDefaultRoute()
	if err != nil {
		return nil, err
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	interfaces := make([]NetworkInterfaceInfo, 0)
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		if ifc.Flags&net.FlagUp == 0 {
			continue
		}

		info, ok, err := buildNetworkInterfaceInfo(ifc, defaultIface, defaultGateway)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		interfaces = append(interfaces, info)
	}

	if len(interfaces) == 0 {
		return nil, fmt.Errorf("no active IPv4 network interfaces found")
	}

	return interfaces, nil
}

func buildNetworkInterfaceInfo(ifc net.Interface, defaultIface, defaultGateway string) (NetworkInterfaceInfo, bool, error) {
	addrs, err := ifc.Addrs()
	if err != nil {
		return NetworkInterfaceInfo{}, false, err
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

		bandwidth, err := GetNetworkBandwidth(ifc.Name)
		if err != nil {
			return NetworkInterfaceInfo{}, false, err
		}

		info := NetworkInterfaceInfo{
			Name:      ifc.Name,
			IP:        ipv4.String(),
			Netmask:   net.IP(ipNet.Mask).String(),
			Bandwidth: bandwidth,
			IsDefault: ifc.Name == defaultIface,
		}
		if info.IsDefault {
			info.Gateway = defaultGateway
		}
		return info, true, nil
	}

	return NetworkInterfaceInfo{}, false, nil
}
