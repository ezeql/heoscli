package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ezeql/heoscli/internal/appconfig"
	"github.com/ezeql/heoscli/internal/heos"
)

// fakeDevice is a HEOS speaker on a loopback port: it answers each
// heos://command?params line from a table and records what it was sent.
type fakeDevice struct {
	addr string

	mu       sync.Mutex
	received []string
	answers  map[string]func(q url.Values) (result, message string, payload any)
}

func startFakeDevice(t *testing.T) *fakeDevice {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	d := &fakeDevice{addr: ln.Addr().String(), answers: map[string]func(url.Values) (string, string, any){}}
	// Two players in the system; the one at 127.0.0.1 is the dialed device.
	d.answer("player/get_players", func(url.Values) (string, string, any) {
		return "success", "", []map[string]any{
			{"pid": 1, "name": "Living Room", "ip": "10.0.0.1"},
			{"pid": 2, "name": "Kitchen", "ip": "127.0.0.1"},
		}
	})
	echo := func(q url.Values) (string, string, any) { return "success", q.Encode(), nil }
	for _, cmd := range []string{"player/set_volume", "player/get_queue", "system/sign_in"} {
		d.answer(cmd, echo)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go d.serve(conn)
		}
	}()
	return d
}

func (d *fakeDevice) answer(cmd string, f func(url.Values) (string, string, any)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.answers[cmd] = f
}

func (d *fakeDevice) serve(conn net.Conn) {
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
		cmd := u.Host + u.Path
		d.mu.Lock()
		d.received = append(d.received, cmd+"?"+u.RawQuery)
		f := d.answers[cmd]
		d.mu.Unlock()
		result, message, payload := "fail", "eid=1&text=unknown command", any(nil)
		if f != nil {
			result, message, payload = f(u.Query())
		}
		out, _ := json.Marshal(map[string]any{
			"heos":    map[string]string{"command": cmd, "result": result, "message": message},
			"payload": payload,
		})
		if _, err := conn.Write(append(out, '\r', '\n')); err != nil {
			return
		}
	}
}

// sent returns the parameters of every cmd the device received.
func (d *fakeDevice) sent(cmd string) []url.Values {
	d.mu.Lock()
	defer d.mu.Unlock()
	var out []url.Values
	for _, r := range d.received {
		if raw, ok := strings.CutPrefix(r, cmd+"?"); ok {
			q, _ := url.ParseQuery(raw)
			out = append(out, q)
		}
	}
	return out
}

// run executes the heos command line with cfg as the stored config.
func run(t *testing.T, cfg appconfig.Config, args ...string) (string, error) {
	t.Helper()
	prev := loadAppConfig
	loadAppConfig = func() (appconfig.Config, error) { return cfg, nil }
	t.Cleanup(func() { loadAppConfig = prev })
	root, err := newRootCmd()
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err = root.ExecuteContext(context.Background())
	return out.String(), err
}

// get_players lists every player in the system: --ip must act on the player
// at that address even when a default device names another one.
func TestIPActsOnThePlayerAtThatAddress(t *testing.T) {
	d := startFakeDevice(t)
	if _, err := run(t, appconfig.Config{DefaultDevice: "Living Room"}, "--ip", d.addr, "volume", "set", "30"); err != nil {
		t.Fatal(err)
	}
	if got := d.sent("player/set_volume"); len(got) != 1 || got[0].Get("pid") != "2" || got[0].Get("level") != "30" {
		t.Errorf("set_volume sent %v, want pid=2 (Kitchen, the player at 127.0.0.1)", got)
	}
}

func TestUnknownPlayerNameFails(t *testing.T) {
	d := startFakeDevice(t)
	// Discovery finds a device called Garage, but no player has that name.
	prev := heosDiscover
	heosDiscover = func(context.Context, heos.DiscoverOptions) ([]heos.Device, error) {
		return []heos.Device{{IP: d.addr, Name: "Garage"}}, nil
	}
	t.Cleanup(func() { heosDiscover = prev })
	if _, err := run(t, appconfig.Config{}, "--name", "Garage", "volume", "set", "30"); err == nil {
		t.Fatal("want an error for a player that does not exist")
	}
	if got := d.sent("player/set_volume"); len(got) != 0 {
		t.Errorf("set_volume sent %v to some other player", got)
	}
}

func TestVolumeOutOfRangeFails(t *testing.T) {
	d := startFakeDevice(t)
	if _, err := run(t, appconfig.Config{}, "--ip", d.addr, "volume", "set", "150"); err == nil {
		t.Fatal("volume 150 must fail")
	}
	if got := d.sent("player/set_volume"); len(got) != 0 {
		t.Errorf("set_volume sent %v", got)
	}
}

func TestStatusReportsDeviceErrors(t *testing.T) {
	d := startFakeDevice(t)
	d.answer("player/get_play_state", func(url.Values) (string, string, any) {
		return "success", "pid=2&state=play", nil
	})
	d.answer("player/get_now_playing_media", func(url.Values) (string, string, any) {
		return "success", "pid=2", map[string]any{"song": "Song"}
	})
	d.answer("player/get_volume", func(url.Values) (string, string, any) {
		return "fail", "eid=2&text=ID Not Valid", nil
	})
	if _, err := run(t, appconfig.Config{}, "--ip", d.addr, "status"); err == nil || !strings.Contains(err.Error(), "ID Not Valid") {
		t.Fatalf("err = %v, want the device's get_volume failure", err)
	}
}

// The HEOS range is start,end inclusive.
func TestQueueMaxIsInclusive(t *testing.T) {
	d := startFakeDevice(t)
	if _, err := run(t, appconfig.Config{}, "--ip", d.addr, "queue", "--max", "5"); err != nil {
		t.Fatal(err)
	}
	if got := d.sent("player/get_queue"); len(got) != 1 || got[0].Get("pid") != "2" || got[0].Get("range") != "0,4" {
		t.Errorf("get_queue sent %v, want range=0,4", got)
	}
}

// Username and password piped into the real binary: one stdin reader must
// serve both prompts, or the first swallows the password line.
func TestPipedSignIn(t *testing.T) {
	d := startFakeDevice(t)
	dir := t.TempDir()
	bin := filepath.Join(dir, "heos")
	if out, err := exec.Command("go", "build", "-o", bin, "github.com/ezeql/heoscli/cmd/heos").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	cmd := exec.Command(bin, "--ip", d.addr, "sign-in")
	cmd.Dir = dir // no .env from the repo
	cmd.Env = append(os.Environ(), "HOME="+dir, "XDG_CONFIG_HOME=", envHEOSUsername+"=", envHEOSPassword+"=")
	cmd.Stdin = strings.NewReader("me@example.com\nhunter2\n")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sign-in: %v\n%s", err, out)
	}
	if got := d.sent("system/sign_in"); len(got) != 1 || got[0].Get("un") != "me@example.com" || got[0].Get("pw") != "hunter2" {
		t.Errorf("sign_in sent %v", got)
	}
}

func TestConfigGetHidesThePassword(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	if _, err := run(t, appconfig.Config{}, "config", "set", "heosPassword", "hunter2"); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"config", "get"},
		{"config", "get", "--format", "json"},
		{"config", "get", "heosPassword"},
	} {
		out, err := run(t, appconfig.Config{}, args...)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, "hunter2") {
			t.Errorf("%v printed the password: %s", args, out)
		}
	}
}
