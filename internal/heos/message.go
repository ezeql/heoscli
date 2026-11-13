package heos

import (
	"encoding/json"
	"net/url"
	"sort"
	"strings"
)

const (
	baseURI   = "heos://"
	separator = "\r\n"
)

// Response is a parsed response from the HEOS device.
type Response struct {
	Command string          `json:"command"`
	Result  string          `json:"result"`
	Message string          `json:"message"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// rawHeosMessage is the JSON envelope from the device.
type rawHeosMessage struct {
	Heos struct {
		Command string `json:"command"`
		Result  string `json:"result"`
		Message string `json:"message"`
	} `json:"heos"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// EncodeCommand builds the HEOS CLI command string: heos://<command>?<query>\r\n
// Params are query-encoded; &, =, % are quoted per spec; url param is last and not quoted.
func EncodeCommand(command string, params map[string]string) string {
	if len(params) == 0 {
		return baseURI + command + separator
	}
	return baseURI + command + "?" + encodeQuery(params) + separator
}

func encodeQuery(params map[string]string) string {
	keys := make([]string, 0, len(params))
	urlVal := ""
	for k := range params {
		if k == AttrURL {
			urlVal = params[k]
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		v := params[k]
		parts = append(parts, k+"="+quoteValue(v))
	}
	if urlVal != "" {
		parts = append(parts, AttrURL+"="+urlVal)
	}
	return strings.Join(parts, "&")
}

func quoteValue(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("%26")
		case '=':
			b.WriteString("%3D")
		case '%':
			b.WriteString("%25")
		case '\r', '\n':
			// A line break would end this command and start another one.
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ParseResponse parses a single line (without \r\n) into Response and message key=value map.
func ParseResponse(line string) (*Response, map[string]string, error) {
	var raw rawHeosMessage
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return nil, nil, err
	}
	msgMap := parseMessageQuery(raw.Heos.Message)
	r := &Response{
		Command: raw.Heos.Command,
		Result:  raw.Heos.Result,
		Message: raw.Heos.Message,
		Payload: raw.Payload,
	}
	return r, msgMap, nil
}

func parseMessageQuery(s string) map[string]string {
	m := make(map[string]string)
	if s == "" {
		return m
	}
	// message is query string: key=value&key2=value2
	vals, err := url.ParseQuery(s)
	if err != nil {
		return m
	}
	for k, v := range vals {
		if len(v) > 0 {
			m[k] = v[0]
		}
	}
	return m
}

// Success returns true if result is success.
func (r *Response) Success() bool {
	return r.Result == ValueSuccess
}
