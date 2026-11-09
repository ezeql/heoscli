package heos

import (
	"context"
	"encoding/json"
	"fmt"
)

// SystemInfo holds system/account info and list of players from get_players.
type SystemInfo struct {
	SignedInUsername string       `json:"signedInUsername,omitempty"`
	Players          []PlayerInfo `json:"players"`
}

// PlayerInfo is one entry from player/get_players.
type PlayerInfo struct {
	PID     int    `json:"pid"`
	Name    string `json:"name"`
	Model   string `json:"model,omitempty"`
	IP      string `json:"ip,omitempty"`
	Serial  string `json:"serial,omitempty"`
	Version string `json:"version,omitempty"`
}

// CheckAccount returns the currently signed-in username, or empty if signed out.
func (c *Client) CheckAccount(ctx context.Context) (string, error) {
	if err := c.Connect(ctx); err != nil {
		return "", err
	}
	_, msgMap, err := c.Command(ctx, CmdAccountCheck, nil)
	if err != nil {
		return "", err
	}
	if msgMap[AttrSignedIn] == "" && msgMap["un"] == "" {
		return "", nil
	}
	return msgMap["un"], nil
}

// SignIn signs in to the HEOS account. Returns the username on success.
func (c *Client) SignIn(ctx context.Context, username, password string) (string, error) {
	if err := c.Connect(ctx); err != nil {
		return "", err
	}
	_, msgMap, err := c.Command(ctx, CmdSignIn, map[string]string{
		AttrUserName: username,
		AttrPassword: password,
	})
	if err != nil {
		return "", err
	}
	u := msgMap["un"]
	if u == "" {
		u = msgMap["username"]
	}
	return u, nil
}

// SignOut signs out of the HEOS account.
func (c *Client) SignOut(ctx context.Context) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}
	_, _, err := c.Command(ctx, CmdSignOut, nil)
	return err
}

// GetSystemInfo returns signed-in user and list of players (from get_players + check_account).
func (c *Client) GetSystemInfo(ctx context.Context) (*SystemInfo, error) {
	if err := c.Connect(ctx); err != nil {
		return nil, err
	}
	username, _ := c.CheckAccount(ctx)
	resp, _, err := c.Command(ctx, CmdGetPlayers, nil)
	if err != nil {
		return nil, err
	}
	var players []PlayerInfo
	if len(resp.Payload) > 0 {
		if err := json.Unmarshal(resp.Payload, &players); err != nil {
			return nil, fmt.Errorf("parse get_players payload: %w", err)
		}
	}
	return &SystemInfo{SignedInUsername: username, Players: players}, nil
}
