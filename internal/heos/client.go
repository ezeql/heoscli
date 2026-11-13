package heos

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Client is a HEOS CLI client for a single host.
type Client struct {
	Host    string
	Timeout time.Duration
	conn    net.Conn
	r       *bufio.Reader // lives as long as conn: bytes past a response are kept
	mu      sync.Mutex
}

// NewClient returns a client for the given host (IP or hostname). Timeout is used for dial and command round-trips.
func NewClient(host string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &Client{Host: host, Timeout: timeout}
}

// Connect establishes a TCP connection to the HEOS device on port 1255.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return nil
	}
	dialer := net.Dialer{Timeout: c.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", hostAddr(c.Host))
	if err != nil {
		return fmt.Errorf("heos connect: %w", err)
	}
	c.conn = conn
	c.r = bufio.NewReader(conn)
	return nil
}

// hostAddr appends the CLI port unless host already carries one.
func hostAddr(host string) string {
	if _, _, err := net.SplitHostPort(host); err == nil {
		return host
	}
	return net.JoinHostPort(host, strconv.Itoa(CLIPort))
}

// Disconnect closes the connection.
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

// Command sends a HEOS command and returns the parsed response and message map. Caller must not be holding c.mu.
func (c *Client) Command(ctx context.Context, command string, params map[string]string) (*Response, map[string]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil, nil, errors.New("heos: not connected")
	}
	line := EncodeCommand(command, params)
	line = strings.TrimSuffix(line, separator)
	if _, err := c.conn.Write([]byte(line + separator)); err != nil {
		_ = c.conn.Close()
		c.conn = nil
		return nil, nil, fmt.Errorf("heos write: %w", err)
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(c.Timeout)
	}
	if err := c.conn.SetReadDeadline(deadline); err != nil {
		return nil, nil, err
	}
	for {
		readLine, err := c.r.ReadString('\n')
		if err != nil {
			_ = c.conn.Close()
			c.conn = nil
			return nil, nil, fmt.Errorf("heos read: %w", err)
		}
		readLine = strings.TrimSuffix(readLine, "\r\n")
		readLine = strings.TrimSuffix(readLine, "\n")
		if readLine == "" {
			continue
		}
		resp, msgMap, err := ParseResponse(readLine)
		if err != nil {
			continue
		}
		if strings.Contains(resp.Message, "command under process") {
			continue
		}
		if strings.HasPrefix(resp.Command, "event/") {
			continue
		}
		if resp.Command != command {
			continue
		}
		if !resp.Success() {
			eid := msgMap[AttrErrorID]
			text := msgMap["text"]
			return resp, msgMap, &CommandError{Command: command, EID: eid, Text: text}
		}
		return resp, msgMap, nil
	}
}

// CommandError is returned when the device reports result != success.
type CommandError struct {
	Command string
	EID     string
	Text    string
}

func (e *CommandError) Error() string {
	if e.Text != "" {
		return e.Command + ": " + e.Text
	}
	if e.EID != "" {
		return e.Command + " error " + e.EID
	}
	return e.Command + " failed"
}
