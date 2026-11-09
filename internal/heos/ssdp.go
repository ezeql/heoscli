package heos

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"net/url"
	"strings"
	"time"
)

// HEOS SSDP search target (Denon HEOS device).
const ssdpST = "urn:schemas-denon-com:device:ACT-Denon:1"

type ssdpResult struct {
	Location string
	USN      string
	ST       string
}

type ssdpUDPConn interface {
	WriteToUDP(b []byte, addr *net.UDPAddr) (int, error)
	ReadFromUDP(b []byte) (int, *net.UDPAddr, error)
	SetReadDeadline(t time.Time) error
	Close() error
}

var ssdpListenUDP = func(network string, laddr *net.UDPAddr) (ssdpUDPConn, error) {
	return net.ListenUDP(network, laddr)
}

var ssdpNow = time.Now

func ssdpDiscover(ctx context.Context, timeout time.Duration) ([]ssdpResult, error) {
	payload := strings.Join([]string{
		"M-SEARCH * HTTP/1.1",
		"HOST: 239.255.255.250:1900",
		`MAN: "ssdp:discover"`,
		"MX: 1",
		"ST: " + ssdpST,
		"", "",
	}, "\r\n")

	conn, err := ssdpListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	dst := &net.UDPAddr{IP: net.ParseIP("239.255.255.250"), Port: 1900}
	for i := 0; i < 3; i++ {
		if _, err := conn.WriteToUDP([]byte(payload), dst); err != nil {
			return nil, err
		}
	}

	deadline := ssdpNow().Add(timeout)
	byLocation := map[string]ssdpResult{}
	buf := make([]byte, 64*1024)

loop:
	for ssdpNow().Before(deadline) {
		select {
		case <-ctx.Done():
			break loop
		default:
		}
		_ = conn.SetReadDeadline(ssdpNow().Add(200 * time.Millisecond))
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				continue
			}
			break loop
		}
		res, ok := parseSSDPResponse(buf[:n])
		if !ok || res.Location == "" {
			continue
		}
		byLocation[res.Location] = res
	}

	out := make([]ssdpResult, 0, len(byLocation))
	for _, v := range byLocation {
		out = append(out, v)
	}
	return out, nil
}

func parseSSDPResponse(b []byte) (ssdpResult, bool) {
	s := bufio.NewScanner(bytes.NewReader(b))
	s.Split(bufio.ScanLines)
	if !s.Scan() {
		return ssdpResult{}, false
	}
	first := strings.TrimSpace(s.Text())
	if !strings.HasPrefix(first, "HTTP/") {
		return ssdpResult{}, false
	}
	headers := map[string]string{}
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			break
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		headers[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}
	return ssdpResult{
		Location: headers["location"],
		USN:      headers["usn"],
		ST:       headers["st"],
	}, true
}

func locationToIP(location string) (string, error) {
	u, err := url.Parse(location)
	if err != nil {
		return "", err
	}
	host := u.Hostname()
	if host == "" {
		return "", nil
	}
	return host, nil
}
