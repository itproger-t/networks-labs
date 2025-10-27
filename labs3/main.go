// go run main.go -host 127.0.0.1 -s 1 -e 65535 -c 200 -timeout 200ms
// go run main.go -host 192.168.1.10 -s 1 -e 65535 -show-procs
package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type result struct {
	Port    int
	Open    bool
	Latency time.Duration
	Proc    string
}

func detectLocalIPv4() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, ifi := range ifaces {
		if ifi.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := ifi.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil {
				continue
			}
			return ip4.String(), nil
		}
	}
	return "127.0.0.1", nil
}

func scanWorker(host string, ports <-chan int, timeout time.Duration, wg *sync.WaitGroup, resCh chan<- result) {
	defer wg.Done()
	for p := range ports {
		addr := net.JoinHostPort(host, strconv.Itoa(p))
		start := time.Now()
		conn, err := net.DialTimeout("tcp", addr, timeout)
		lat := time.Since(start)
		if err == nil {
			_ = conn.Close()
			resCh <- result{Port: p, Open: true, Latency: lat}
		} else {
			resCh <- result{Port: p, Open: false, Latency: lat}
		}
	}
}

func parseLsofListen() map[int]string {
	out := map[int]string{}

	_, err := exec.LookPath("lsof")
	if err != nil {
		return out
	}

	cmd := exec.Command("lsof", "-nP", "-iTCP", "-sTCP:LISTEN", "-F", "pcfn")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return out
	}
	if err := cmd.Start(); err != nil {
		return out
	}
	scanner := bufio.NewScanner(stdout)
	var curPid, curCmd string
	portRe := regexp.MustCompile(`:(\d+)$`)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		switch line[0] {
		case 'p':
			curPid = strings.TrimPrefix(line, "p")
		case 'c':
			curCmd = strings.TrimPrefix(line, "c")
		case 'n':
			n := strings.TrimPrefix(line, "n")
			m := portRe.FindStringSubmatch(n)
			if len(m) == 2 {
				if port, err := strconv.Atoi(m[1]); err == nil {
					procStr := curCmd
					if curPid != "" {
						procStr = fmt.Sprintf("%s(%s)", procStr, curPid)
					}
					out[port] = procStr
				}
			}
		}
	}
	_ = cmd.Wait()
	return out
}

func main() {
	hostFlag := flag.String("host", "", "host to scan (if empty, will attempt to detect local IPv4)")
	start := flag.Int("s", 1, "start port")
	end := flag.Int("e", 1024, "end port")
	concurrency := flag.Int("c", 200, "concurrency (number of workers)")
	timeoutFlag := flag.Duration("timeout", 300*time.Millisecond, "dial timeout per port (e.g. 300ms)")
	showProcs := flag.Bool("show-procs", false, "attempt to show process owning listening ports (uses lsof)")
	flag.Parse()

	host := *hostFlag
	if host == "" {
		ip, err := detectLocalIPv4()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot detect local IP: %v\n", err)
			os.Exit(1)
		}
		host = ip
	}

	if *start < 1 {
		*start = 1
	}
	if *end > 65535 {
		*end = 65535
	}
	if *end < *start {
		fmt.Fprintf(os.Stderr, "end port must be >= start port\n")
		os.Exit(1)
	}

	total := *end - *start + 1
	fmt.Printf("Scanning %s ports %d..%d (total %d) concurrency=%d timeout=%v\n", host, *start, *end, total, *concurrency, *timeoutFlag)

	var procMap map[int]string
	if *showProcs {
		fmt.Println("Gathering listening processes via lsof...")
		procMap = parseLsofListen()
		if len(procMap) == 0 {
			fmt.Println("Warning: could not get process map from lsof (lsof missing or permission).")
		}
	}

	ports := make(chan int, *concurrency)
	resCh := make(chan result, total)

	var wg sync.WaitGroup
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go scanWorker(host, ports, *timeoutFlag, &wg, resCh)
	}

	// feed ports
	go func() {
		for p := *start; p <= *end; p++ {
			ports <- p
		}
		close(ports)
	}()

	go func() {
		wg.Wait()
		close(resCh)
	}()

	results := make([]result, 0, total)
	for r := range resCh {
		if *showProcs && r.Open {
			if proc, ok := procMap[r.Port]; ok {
				r.Proc = proc
			}
		}
		results = append(results, r)
	}

	// sort by port
	sort.Slice(results, func(i, j int) bool { return results[i].Port < results[j].Port })

	openCount := 0
	for _, r := range results {
		if r.Open {
			openCount++
			if r.Proc != "" {
				fmt.Printf("Port %5d OPEN  (latency %v) proc=%s\n", r.Port, r.Latency, r.Proc)
			} else {
				fmt.Printf("Port %5d OPEN  (latency %v)\n", r.Port, r.Latency)
			}
		}
	}

	fmt.Printf("\nScan complete: %d open ports (scanned %d ports)\n", openCount, len(results))
}
