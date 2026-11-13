package heos

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// PlayState is the player state (play, pause, stop).
const (
	PlayStatePlay  = "play"
	PlayStatePause = "pause"
	PlayStateStop  = "stop"
)

// GetPlayers returns all players (same as system get_players). Convenience that parses into []PlayerInfo.
func (c *Client) GetPlayers(ctx context.Context) ([]PlayerInfo, error) {
	info, err := c.GetSystemInfo(ctx)
	if err != nil {
		return nil, err
	}
	return info.Players, nil
}

// GetPlayState returns the current play state for a player.
func (c *Client) GetPlayState(ctx context.Context, playerID int) (string, error) {
	if err := c.Connect(ctx); err != nil {
		return "", err
	}
	_, msgMap, err := c.Command(ctx, CmdGetPlayState, map[string]string{
		AttrPlayerID: strconv.Itoa(playerID),
	})
	if err != nil {
		return "", err
	}
	return msgMap[AttrState], nil
}

// SetPlayState sets play state (play, pause, stop).
func (c *Client) SetPlayState(ctx context.Context, playerID int, state string) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}
	_, _, err := c.Command(ctx, CmdSetPlayState, map[string]string{
		AttrPlayerID: strconv.Itoa(playerID),
		AttrState:    state,
	})
	return err
}

// NowPlayingMedia is the now playing info from get_now_playing_media.
type NowPlayingMedia struct {
	Type     string `json:"type"`
	Song     string `json:"song,omitempty"`
	Album    string `json:"album,omitempty"`
	Artist   string `json:"artist,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	MediaID  string `json:"mid,omitempty"`
	QueueID  int    `json:"qid,omitempty"`
	Quality  string `json:"quality,omitempty"`
}

// GetNowPlayingMedia returns the currently playing media for a player.
func (c *Client) GetNowPlayingMedia(ctx context.Context, playerID int) (*NowPlayingMedia, error) {
	if err := c.Connect(ctx); err != nil {
		return nil, err
	}
	resp, _, err := c.Command(ctx, CmdGetNowPlayingMedia, map[string]string{
		AttrPlayerID: strconv.Itoa(playerID),
	})
	if err != nil {
		return nil, err
	}
	out := &NowPlayingMedia{}
	if len(resp.Payload) == 0 {
		return out, nil
	}
	var raw map[string]any
	if err := json.Unmarshal(resp.Payload, &raw); err != nil {
		_ = json.Unmarshal(resp.Payload, out)
		return out, nil
	}
	out.Type = getStr(raw, "type")
	out.Song = getStr(raw, "song")
	out.Album = getStr(raw, "album")
	out.Artist = getStr(raw, "artist")
	out.ImageURL = getStr(raw, "image_url")
	out.MediaID = getStr(raw, "mid")
	out.QueueID = toInt(raw["qid"])
	out.Quality = getStr(raw, "quality")
	if out.Quality == "" {
		out.Quality = getStr(raw, "audio_quality")
	}
	if out.Quality == "" {
		out.Quality = getStr(raw, "streaming_quality")
	}
	if out.Quality == "" {
		out.Quality = getStr(raw, "sound_quality")
	}
	if out.Quality == "" {
		out.Quality = getStr(raw, "bit_depth")
	}
	if out.Quality == "" {
		out.Quality = getStr(raw, "sample_rate")
	}
	return out, nil
}

// QueueItem is one entry from player/get_queue (current play queue).
type QueueItem struct {
	QueueID  int    `json:"qid"`
	MediaID  string `json:"mid,omitempty"`
	Name     string `json:"name,omitempty"`
	Song     string `json:"song,omitempty"`
	Artist   string `json:"artist,omitempty"`
	Album    string `json:"album,omitempty"`
	Type     string `json:"type,omitempty"`
	SourceID int    `json:"sid,omitempty"`
	Quality  string `json:"quality,omitempty"`
}

// GetQueue returns the current play queue for a player (tracks from any source, e.g. SoundCloud, Tidal).
// start is the 0-based index, count is how many items to return (e.g. 0, 100 for first 100).
func (c *Client) GetQueue(ctx context.Context, playerID int, start, count int) ([]QueueItem, error) {
	if err := c.Connect(ctx); err != nil {
		return nil, err
	}
	params := map[string]string{
		AttrPlayerID: strconv.Itoa(playerID),
	}
	if count > 0 {
		// The range is inclusive: start,end.
		params[AttrRange] = fmt.Sprintf("%d,%d", start, start+count-1)
	}
	resp, _, err := c.Command(ctx, CmdGetQueue, params)
	if err != nil {
		return nil, err
	}
	var list []QueueItem
	if len(resp.Payload) == 0 {
		return list, nil
	}
	var raw []map[string]any
	if err := json.Unmarshal(resp.Payload, &raw); err != nil {
		return nil, fmt.Errorf("parse get_queue payload: %w", err)
	}
	for _, m := range raw {
		qid := toInt(m["qid"])
		sid := toInt(m["sid"])
		name := getStr(m, "name")
		if name == "" {
			name = getStr(m, "song")
		}
		quality := getStr(m, "quality")
		if quality == "" {
			quality = getStr(m, "audio_quality")
		}
		if quality == "" {
			quality = getStr(m, "streaming_quality")
		}
		if quality == "" {
			quality = getStr(m, "sound_quality")
		}
		if quality == "" {
			quality = getStr(m, "bit_depth")
		}
		if quality == "" {
			quality = getStr(m, "sample_rate")
		}
		list = append(list, QueueItem{
			QueueID:  qid,
			MediaID:  getStr(m, "mid"),
			Name:     name,
			Song:     getStr(m, "song"),
			Artist:   getStr(m, "artist"),
			Album:    getStr(m, "album"),
			Type:     getStr(m, "type"),
			SourceID: sid,
			Quality:  quality,
		})
	}
	return list, nil
}

// GetVolume returns volume 0-100 for a player.
func (c *Client) GetVolume(ctx context.Context, playerID int) (int, error) {
	if err := c.Connect(ctx); err != nil {
		return 0, err
	}
	_, msgMap, err := c.Command(ctx, CmdGetVolume, map[string]string{
		AttrPlayerID: strconv.Itoa(playerID),
	})
	if err != nil {
		return 0, err
	}
	v, _ := strconv.Atoi(msgMap[AttrLevel])
	return v, nil
}

// SetVolume sets volume 0-100.
func (c *Client) SetVolume(ctx context.Context, playerID int, level int) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}
	if level < 0 || level > 100 {
		return fmt.Errorf("volume %d out of range 0-100", level)
	}
	_, _, err := c.Command(ctx, CmdSetVolume, map[string]string{
		AttrPlayerID: strconv.Itoa(playerID),
		AttrLevel:    strconv.Itoa(level),
	})
	return err
}

// VolumeUp increases volume by step (1-10).
func (c *Client) VolumeUp(ctx context.Context, playerID int, step int) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}
	if step <= 0 {
		step = 5
	}
	if step > 10 {
		step = 10
	}
	_, _, err := c.Command(ctx, CmdVolumeUp, map[string]string{
		AttrPlayerID: strconv.Itoa(playerID),
		AttrStep:     strconv.Itoa(step),
	})
	return err
}

// VolumeDown decreases volume by step (1-10).
func (c *Client) VolumeDown(ctx context.Context, playerID int, step int) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}
	if step <= 0 {
		step = 5
	}
	if step > 10 {
		step = 10
	}
	_, _, err := c.Command(ctx, CmdVolumeDown, map[string]string{
		AttrPlayerID: strconv.Itoa(playerID),
		AttrStep:     strconv.Itoa(step),
	})
	return err
}

// GetMute returns whether the player is muted.
func (c *Client) GetMute(ctx context.Context, playerID int) (bool, error) {
	if err := c.Connect(ctx); err != nil {
		return false, err
	}
	_, msgMap, err := c.Command(ctx, CmdGetMute, map[string]string{
		AttrPlayerID: strconv.Itoa(playerID),
	})
	if err != nil {
		return false, err
	}
	return msgMap[AttrState] == ValueOn, nil
}

// SetMute sets mute on or off.
func (c *Client) SetMute(ctx context.Context, playerID int, mute bool) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}
	v := ValueOff
	if mute {
		v = ValueOn
	}
	_, _, err := c.Command(ctx, CmdSetMute, map[string]string{
		AttrPlayerID: strconv.Itoa(playerID),
		AttrState:    v,
	})
	return err
}

// PlayNext skips to next track.
func (c *Client) PlayNext(ctx context.Context, playerID int) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}
	_, _, err := c.Command(ctx, CmdPlayNext, map[string]string{
		AttrPlayerID: strconv.Itoa(playerID),
	})
	return err
}

// PlayPrevious goes to previous track.
func (c *Client) PlayPrevious(ctx context.Context, playerID int) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}
	_, _, err := c.Command(ctx, CmdPlayPrevious, map[string]string{
		AttrPlayerID: strconv.Itoa(playerID),
	})
	return err
}
