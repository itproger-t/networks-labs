package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	cidr    = flag.String("cidr", "", "Network in CIDR notation, e.g. 192.168.1.0/24 (required)")
	ports   = flag.String("ports", "22,80,443,3389,445,139,135", "Comma-separated list of ports to scan")
	timeout = flag.Duration("timeout", 1000*time.Millisecond, "Dial timeout per port")
	workers = flag.Int("workers", 500, "Max concurrent goroutines (global)")
)

func main() {
	flag.Parse()
	if *cidr == "" {
		fmt.Println("Usage: go run scanner.go -cidr 192.168.1.0/24 [-ports 22,80] [-timeout 500ms] [-workers 200]")
		return
	}

	portList, err := parsePorts(*ports)
	if err != nil {
		fmt.Println("Invalid ports:", err)
		return
	}

	_, ipnet, err := net.ParseCIDR(*cidr)
	if err != nil {
		fmt.Println("Invalid CIDR:", err)
		return
	}

	ips := ipsFromCIDR(ipnet)

	fmt.Printf("Scanning %d addresses, %d ports each (timeout %s)\n", len(ips), len(portList), (*timeout).String())

	sem := make(chan struct{}, *workers) // semaphore controlling concurrency
	var wg sync.WaitGroup
	results := make(map[string][]int)
	var resMu sync.Mutex

	for _, ip := range ips {
		ipStr := ip.String()
		wg.Add(1)
		sem <- struct{}{}
		go func(target string) {
			defer wg.Done()
			defer func() { <-sem }()
			open := scanPortsOnHost(target, portList, *timeout)
			if len(open) > 0 {
				resMu.Lock()
				results[target] = open
				resMu.Unlock()
				fmt.Printf("%-15s open: %v\n", target, open)
			}
		}(ipStr)
	}

	wg.Wait()

	fmt.Println("\n=== Summary ===")
	if len(results) == 0 {
		fmt.Println("No open ports found in the scanned range (within given ports/timeouts).")
		return
	}
	for ip, ps := range results {
		fmt.Printf("%-15s -> %v\n", ip, ps)
	}
}

// parsePorts parses string like "22,80,443" into []int
func parsePorts(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, err
		}
		if n < 1 || n > 65535 {
			return nil, fmt.Errorf("port out of range: %d", n)
		}
		out = append(out, n)
	}
	return out, nil
}

func ipsFromCIDR(ipnet *net.IPNet) []net.IP {
	var ips []net.IP
	network := ipnet.IP.Mask(ipnet.Mask).To4()
	if network == nil {
		return ips
	}
	mask := net.IP(ipnet.Mask).To4()
	if mask == nil {
		return ips
	}

	start := make(net.IP, 4)
	copy(start, network)

	broadcast := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		broadcast[i] = network[i] | (^mask[i])
	}

	for ip := make(net.IP, 4); ; incrementIP(start) {
		copy(ip, start)
		ips = append(ips, append(net.IP(nil), ip...))
		if start.Equal(broadcast) {
			break
		}
	}
	return ips
}

func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

func scanPortsOnHost(target string, ports []int, timeout time.Duration) []int {
	var mu sync.Mutex
	var wg sync.WaitGroup
	open := make([]int, 0)
	hostSem := make(chan struct{}, 100)

	for _, p := range ports {
		wg.Add(1)
		hostSem <- struct{}{}
		go func(port int) {
			defer wg.Done()
			defer func() { <-hostSem }()
			addr := fmt.Sprintf("%s:%d", target, port)
			conn, err := net.DialTimeout("tcp", addr, timeout)
			if err == nil {
				conn.Close()
				mu.Lock()
				open = append(open, port)
				mu.Unlock()
			}
		}(p)
	}
	wg.Wait()
	return open
}
