package heos

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"net/url"
	"strings"
	"testing"
	"time"
)

// Every device answers get_players with the whole system, so each one must
// be named after the player at its own address, not the first in the list.
func TestEnrichDeviceNamesUsesThePlayerAtEachAddress(t *testing.T) {
	var listeners []net.Listener
	var devs []Device
	for range 2 {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = ln.Close() })
		listeners = append(listeners, ln)
		devs = append(devs, Device{IP: ln.Addr().String()})
	}
	players := []PlayerInfo{{PID: 1, Name: "Living Room", IP: devs[0].IP}, {PID: 2, Name: "Kitchen", IP: devs[1].IP}}
	for _, ln := range listeners {
		go servePlayers(ln, players)
	}
	enrichDeviceNames(context.Background(), devs, time.Second)
	if devs[0].Name != "Living Room" || devs[1].Name != "Kitchen" {
		t.Errorf("names = %q, %q", devs[0].Name, devs[1].Name)
	}
}

func servePlayers(ln net.Listener, players []PlayerInfo) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go func() {
			defer func() { _ = conn.Close() }()
			r := bufio.NewReader(conn)
			for {
				line, err := r.ReadString('\n')
				if err != nil {
					return
				}
				u, err := url.Parse(strings.TrimSpace(line))
				if err != nil {
					return
				}
				reply := map[string]any{"heos": map[string]string{"command": u.Host + u.Path, "result": "success", "message": ""}}
				if u.Host+u.Path == CmdGetPlayers {
					reply["payload"] = players
				}
				out, _ := json.Marshal(reply)
				if _, err := conn.Write(append(out, '\r', '\n')); err != nil {
					return
				}
			}
		}()
	}
}
