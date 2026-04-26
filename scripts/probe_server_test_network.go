package main
package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	loopbackStatus, loopbackReason := probeListen("tcp4", "127.0.0.1:0", false)
	ipv6Status, ipv6Reason := probeListen("tcp6", "[::1]:0", true)
	hostStatus, hostReason := probeHostInterfaces()

	networkStatus := "ok"
	if loopbackStatus != "ok" || hostStatus != "ok" || ipv6Status == "blocked" {
		networkStatus = "blocked"
	}

	fmt.Printf("loopback_listener=%s\n", loopbackStatus)
	fmt.Printf("loopback_listener_reason=%s\n", sanitizeReason(loopbackReason))
	fmt.Printf("ipv6_loopback_listener=%s\n", ipv6Status)
	fmt.Printf("ipv6_loopback_listener_reason=%s\n", sanitizeReason(ipv6Reason))
	fmt.Printf("host_interfaces=%s\n", hostStatus)
	fmt.Printf("host_interfaces_reason=%s\n", sanitizeReason(hostReason))
	fmt.Printf("network_sensitive_tests=%s\n", networkStatus)
}

func probeListen(network, addr string, allowUnavailable bool) (string, string) {
	listener, err := net.Listen(network, addr)
	if err == nil {
		_ = listener.Close()
		return "ok", ""
	}

	message := strings.ToLower(err.Error())
	if strings.Contains(message, "operation not permitted") ||
		strings.Contains(message, "permission denied") ||
		strings.Contains(message, "not permitted") {
		return "blocked", err.Error()
	}

	if allowUnavailable &&
		(strings.Contains(message, "address family not supported") ||
			strings.Contains(message, "cannot assign requested address") ||
			strings.Contains(message, "no suitable address")) {
		return "unavailable", err.Error()
	}

	if allowUnavailable {
		return "blocked", err.Error()
	}
	return "blocked", err.Error()
}

func probeHostInterfaces() (string, string) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "blocked", err.Error()
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip := ipFromAddr(addr)
			if ip == nil || ip.IsLoopback() || ip.IsUnspecified() {
				continue
			}
			return "ok", ""
		}
	}

	return "blocked", "no_usable_non_loopback_interface"
}

func ipFromAddr(addr net.Addr) net.IP {
	switch typed := addr.(type) {
	case *net.IPNet:
		return typed.IP
	case *net.IPAddr:
		return typed.IP
	default:
		return nil
	}
}

func sanitizeReason(reason string) string {
	if strings.TrimSpace(reason) == "" {
		return "none"
	}
	replacer := strings.NewReplacer(
		"\n", " ",
		"\r", " ",
		"\t", " ",
		"=", ":",
	)
	clean := strings.TrimSpace(replacer.Replace(reason))
	clean = strings.Join(strings.Fields(clean), "_")
	if clean == "" {
		return "none"
	}
	if len(clean) > 240 {
		return clean[:240]
	}
	return clean
}

func init() {
	if _, err := os.Stat("."); err != nil {
		panic(err)
	}
}
