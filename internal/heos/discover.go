package heos

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Device is a discovered HEOS device.
type Device struct {
	IP       string `json:"ip"`
	Name     string `json:"name"`
	UDN      string `json:"udn,omitempty"`
	Location string `json:"location,omitempty"`
}

// DiscoverOptions configures discovery.
type DiscoverOptions struct {
	Timeout time.Duration
}

var (
	ssdpDiscoverFunc = ssdpDiscover
	scanPort1255Func = scanPort1255
	locationToIPFunc = locationToIP
)

// Discover finds HEOS devices on the local network via SSDP, with optional port-scan fallback.
// Each device's Name is filled by querying the device (get_players) when possible.
func Discover(ctx context.Context, opts DiscoverOptions) ([]Device, error) {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	opCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ssdpTimeout := 1500 * time.Millisecond
	if timeout <= 2*time.Second {
		ssdpTimeout = timeout / 2
	}
	ssdpCtx, cancelSSDP := context.WithTimeout(opCtx, ssdpTimeout)
	ssdpResults, err := ssdpDiscoverFunc(ssdpCtx, ssdpTimeout)
	cancelSSDP()
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return nil, err
	}

	byIP := map[string]Device{}
	for _, r := range ssdpResults {
		if r.Location == "" {
			continue
		}
		ip, err := locationToIPFunc(r.Location)
		if err != nil || ip == "" {
			continue
		}
		name := ip
		byIP[ip] = Device{IP: ip, Name: name, UDN: r.USN, Location: r.Location}
	}

	if len(byIP) == 0 {
		// Fallback: scan local subnets for port 1255.
		if ip, scanErr := scanPort1255Func(opCtx, timeout); scanErr == nil && ip != "" {
			byIP[ip] = Device{IP: ip, Name: ip}
		}
	}

	if len(byIP) == 0 {
		return nil, errors.New("no HEOS devices found")
	}

	devs := sortDevices(byIP)
	enrichDeviceNames(opCtx, devs, timeout)
	return devs, nil
}

func sortDevices(byIP map[string]Device) []Device {
	out := make([]Device, 0, len(byIP))
	for _, d := range byIP {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].IP < out[j].IP
	})
	return out
}

// enrichDeviceNames sets each device's Name from get_players when possible (e.g. "Living Room").
func enrichDeviceNames(ctx context.Context, devs []Device, timeout time.Duration) {
	if len(devs) == 0 {
		return
	}
	enrichTimeout := min(timeout, 2*time.Second)
	type result struct {
		index int
		name  string
	}
	ch := make(chan result, len(devs))
	for i := range devs {
		go func() {
			name := devs[i].IP
			c := NewClient(devs[i].IP, enrichTimeout)
			info, err := c.GetSystemInfo(ctx)
			_ = c.Disconnect()
			// get_players lists the whole system; pick the player at this IP.
			if err == nil {
				for _, p := range info.Players {
					if p.IP == devs[i].IP && strings.TrimSpace(p.Name) != "" {
						name = strings.TrimSpace(p.Name)
						break
					}
				}
			}
			ch <- result{index: i, name: name}
		}()
	}
	for range devs {
		r := <-ch
		devs[r.index].Name = r.name
	}
}

// scanPort1255 scans local /24 subnets for hosts with port 1255 open. Returns first found IP.
func scanPort1255(ctx context.Context, timeout time.Duration) (string, error) {
	addrs, err := localIPv4Addrs()
	if err != nil || len(addrs) == 0 {
		return "", errors.New("no local IPv4 addresses")
	}
	candidateIPs := make(chan string, 512)
	found := make(chan string, 1)
	workers := 64
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for ip := range candidateIPs {
				select {
				case <-ctx.Done():
					return
				default:
				}
				if !isPortOpen(ip, CLIPort, 250*time.Millisecond) {
					continue
				}
				select {
				case found <- ip:
				default:
				}
				return
			}
		}()
	}

	seen := map[string]struct{}{}
	for _, ip := range addrs {
		prefix := ipTo24Prefix(ip)
		if prefix == "" {
			continue
		}
		if _, ok := seen[prefix]; ok {
			continue
		}
		seen[prefix] = struct{}{}
		for host := 1; host <= 254; host++ {
			select {
			case candidateIPs <- fmt.Sprintf("%s.%d", prefix, host):
			case ip := <-found:
				close(candidateIPs)
				wg.Wait()
				return ip, nil
			case <-ctx.Done():
				close(candidateIPs)
				wg.Wait()
				return "", ctx.Err()
			}
		}
	}
	close(candidateIPs)
	wg.Wait()
	select {
	case ip := <-found:
		return ip, nil
	default:
		return "", errors.New("no HEOS devices found on local subnets")
	}
}

func localIPv4Addrs() ([]net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var ips []net.IP
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipNet, ok := a.(*net.IPNet)
			if !ok || ipNet.IP == nil {
				continue
			}
			ip4 := ipNet.IP.To4()
			if ip4 == nil {
				continue
			}
			ips = append(ips, ip4)
		}
	}
	return ips, nil
}

func ipTo24Prefix(ip net.IP) string {
	ip4 := ip.To4()
	if ip4 == nil {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d", ip4[0], ip4[1], ip4[2])
}

func isPortOpen(ip string, port int, timeout time.Duration) bool {
	d := net.Dialer{Timeout: timeout}
	conn, err := d.Dial("tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
