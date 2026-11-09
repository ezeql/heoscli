package heos

import (
	"encoding/json"
	"testing"
)

func TestEncodeCommand_noParams(t *testing.T) {
	out := EncodeCommand("system/heart_beat", nil)
	want := "heos://system/heart_beat\r\n"
	if out != want {
		t.Errorf("EncodeCommand() = %q, want %q", out, want)
	}
}

func TestEncodeCommand_withParams(t *testing.T) {
	params := map[string]string{"pid": "1", "state": "play"}
	out := EncodeCommand("player/set_play_state", params)
	if out != "heos://player/set_play_state?pid=1&state=play\r\n" &&
		out != "heos://player/set_play_state?state=play&pid=1\r\n" {
		t.Errorf("EncodeCommand() = %q (order may vary)", out)
	}
}

func TestQuoteValue(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"a&b", "a%26b"},
		{"x=y", "x%3Dy"},
		{"%", "%25"},
		{"normal", "normal"},
	}
	for _, tt := range tests {
		got := quoteValue(tt.in)
		if got != tt.want {
			t.Errorf("quoteValue(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseResponse_success(t *testing.T) {
	line := `{"heos":{"command":"player/get_volume","result":"success","message":"pid=1&level=50"},"payload":{}}`
	resp, msgMap, err := ParseResponse(line)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Command != "player/get_volume" {
		t.Errorf("Command = %q", resp.Command)
	}
	if !resp.Success() {
		t.Error("Success() = false")
	}
	if msgMap["pid"] != "1" || msgMap["level"] != "50" {
		t.Errorf("message map = %v", msgMap)
	}
}

func TestParseResponse_fail(t *testing.T) {
	line := `{"heos":{"command":"system/sign_in","result":"fail","message":"eid=6&text=Invalid%20credentials"},"payload":{}}`
	resp, msgMap, err := ParseResponse(line)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Success() {
		t.Error("Success() = true")
	}
	if msgMap[AttrErrorID] != "6" {
		t.Errorf("eid = %q", msgMap[AttrErrorID])
	}
}

func TestParseResponse_payloadArray(t *testing.T) {
	line := `{"heos":{"command":"player/get_players","result":"success","message":""},"payload":[{"pid":"1","name":"Living Room","model":"HEOS 1"}]}`
	resp, _, err := ParseResponse(line)
	if err != nil {
		t.Fatal(err)
	}
	var list []map[string]string
	if err := json.Unmarshal(resp.Payload, &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0]["name"] != "Living Room" {
		t.Errorf("payload = %v", list)
	}
}
