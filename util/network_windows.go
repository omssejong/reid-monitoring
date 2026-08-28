//go:build windows

package util

import (
	"fmt"
	"net"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modIphlpapiNet = windows.NewLazySystemDLL("iphlpapi.dll")

	// GetIfEntry2 는 Vista 이상에서 제공된다. x/sys/windows 는 Win8 이상 전용인
	// GetIfEntry2Ex 만 노출하므로 여기서 직접 바인딩한다.
	procGetIfEntry2 = modIphlpapiNet.NewProc("GetIfEntry2")
)

// adapterBufferSize GetAdaptersAddresses 초기 버퍼 크기. MSDN 권장값 15KB.
const adapterBufferSize = 15 * 1024

// GetNetworkBandwidth 어댑터의 송신 링크 속도(bps)를 Mb/s 문자열로 반환한다.
// (리눅스의 /sys/class/net/<iface>/speed 대응)
func GetNetworkBandwidth(interfaceName string) (string, error) {
	adapter, err := findAdapterByName(interfaceName)
	if err != nil || adapter == nil {
		return "unknown", nil
	}

	speedBps := adapter.TransmitLinkSpeed
	// 드라이버가 속도를 모르면 0 또는 ^uint64(0) 을 반환한다.
	if speedBps == 0 || speedBps == ^uint64(0) {
		return "unknown", nil
	}

	speedMbps := speedBps / 1000 / 1000
	if speedMbps == 0 {
		return "unknown", nil
	}

	return fmt.Sprintf("%dMb/s", speedMbps), nil
}

// GetNetworkUsage GetIfEntry2 로 인터페이스 누적 송수신 바이트를 조회한다.
// (리눅스의 /proc/net/dev 대응)
//
// 대상 인터페이스를 찾지 못하거나 카운터 조회에 실패해도 error 대신 0을 반환한다.
// /proc/net/dev 를 훑는 리눅스판이 일치하는 줄이 없으면 조용히 0을 돌려주는 것과 같은 동작으로,
// networkName 오타 하나 때문에 buildSnapshot 전체가 실패해 CPU/GPU/메모리 지표까지
// 멈추는 것을 막는다. 다만 설정 실수가 묻히지 않도록 최초 1회 경고를 남긴다.
func GetNetworkUsage(interfaceName string) (NetworkUsage, error) {
	ifc, err := net.InterfaceByName(interfaceName)
	if err != nil {
		warnNetworkUsageOnce(fmt.Sprintf("네트워크 인터페이스 %q 를 찾을 수 없어 대역폭을 0으로 보고합니다 (config networkName 확인): %v", interfaceName, err))
		return NetworkUsage{}, nil
	}

	var row windows.MibIfRow2
	row.InterfaceIndex = uint32(ifc.Index)

	// GetIfEntry2 는 NETIO_STATUS 를 반환한다 (NO_ERROR == 0).
	if r, _, _ := procGetIfEntry2.Call(uintptr(unsafe.Pointer(&row))); r != 0 {
		warnNetworkUsageOnce(fmt.Sprintf("GetIfEntry2(%s) 실패로 대역폭을 0으로 보고합니다: %v", interfaceName, windows.Errno(r)))
		return NetworkUsage{}, nil
	}

	return NetworkUsage{
		bytesRecv: row.InOctets,
		bytesSent: row.OutOctets,
	}, nil
}

// getDefaultRoute 기본 게이트웨이를 가진 인터페이스 이름과 게이트웨이 IPv4 주소를 반환한다.
// 리눅스의 /proc/net/route 대신 어댑터 목록에서 게이트웨이가 설정된 항목 중
// IPv4 메트릭이 가장 낮은(= 우선순위가 가장 높은) 어댑터를 고른다.
func getDefaultRoute() (string, string, error) {
	adapters, err := adapterAddresses()
	if err != nil {
		return "", "", err
	}

	bestName := ""
	bestGateway := ""
	bestMetric := ^uint32(0)

	for _, adapter := range adapters {
		if adapter.OperStatus != windows.IfOperStatusUp {
			continue
		}

		gateway := firstIPv4Gateway(adapter)
		if gateway == "" {
			continue
		}

		if bestName == "" || adapter.Ipv4Metric < bestMetric {
			bestName = windows.UTF16PtrToString(adapter.FriendlyName)
			bestGateway = gateway
			bestMetric = adapter.Ipv4Metric
		}
	}

	// 게이트웨이가 없는 폐쇄망 구성도 정상 동작해야 하므로 에러 대신 빈 값을 돌려준다.
	// (리눅스판도 default route 가 없으면 빈 문자열을 반환한다)
	return bestName, bestGateway, nil
}

// firstIPv4Gateway 어댑터에 설정된 첫 번째 IPv4 게이트웨이 주소를 반환한다.
func firstIPv4Gateway(adapter *windows.IpAdapterAddresses) string {
	for gw := adapter.FirstGatewayAddress; gw != nil; gw = gw.Next {
		ip := gw.Address.IP()
		if ip == nil {
			continue
		}
		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4.String()
		}
	}
	return ""
}

// findAdapterByName net.Interface.Name(윈도우에서는 어댑터 표시 이름)으로 어댑터를 찾는다.
func findAdapterByName(interfaceName string) (*windows.IpAdapterAddresses, error) {
	adapters, err := adapterAddresses()
	if err != nil {
		return nil, err
	}

	for _, adapter := range adapters {
		if windows.UTF16PtrToString(adapter.FriendlyName) == interfaceName {
			return adapter, nil
		}
	}
	return nil, nil
}

// adapterAddresses GetAdaptersAddresses 결과를 슬라이스로 펼쳐 반환한다.
// 반환된 포인터들은 내부 버퍼를 가리키며, GC 는 내부 포인터도 추적하므로 안전하다.
func adapterAddresses() ([]*windows.IpAdapterAddresses, error) {
	size := uint32(adapterBufferSize)

	// 어댑터 구성이 조회 도중 바뀌면 버퍼가 다시 모자랄 수 있어 몇 번 재시도한다.
	for attempt := 0; attempt < 5; attempt++ {
		buf := make([]byte, size)
		head := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))

		err := windows.GetAdaptersAddresses(
			windows.AF_UNSPEC,
			windows.GAA_FLAG_INCLUDE_GATEWAYS|windows.GAA_FLAG_SKIP_ANYCAST|windows.GAA_FLAG_SKIP_MULTICAST|windows.GAA_FLAG_SKIP_DNS_SERVER,
			0,
			head,
			&size,
		)
		if err == windows.ERROR_BUFFER_OVERFLOW {
			continue
		}
		if err != nil {
			return nil, err
		}

		adapters := make([]*windows.IpAdapterAddresses, 0)
		for adapter := head; adapter != nil; adapter = adapter.Next {
			adapters = append(adapters, adapter)
		}
		return adapters, nil
	}

	return nil, fmt.Errorf("GetAdaptersAddresses: buffer size not settled")
}

// networkUsageWarnOnce 대역폭 수집 실패 경고를 1초 주기 수집 루프에서 반복 출력하지 않도록 1회로 제한한다.
var networkUsageWarnOnce sync.Once

func warnNetworkUsageOnce(message string) {
	networkUsageWarnOnce.Do(func() { log.Warn(message) })
}
