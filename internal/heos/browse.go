package heos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"unicode/utf8"
)

// MusicSource is a music source from get_music_sources.
type MusicSource struct {
	SID             int    `json:"sid"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	Available       bool   `json:"available"`
	ImageURL        string `json:"image_url,omitempty"`
	ServiceUsername string `json:"service_username,omitempty"`
}

// SearchCriteria is one search criterion from get_search_criteria.
type SearchCriteria struct {
	Name     string `json:"name"`
	Scid     int    `json:"scid"`
	Wildcard string `json:"wildcard,omitempty"`
	Cid      string `json:"cid,omitempty"`
	Playable string `json:"playable,omitempty"`
}

// MediaItem is a search or browse result item (song, album, etc.).
type MediaItem struct {
	SourceID    int    `json:"-"`
	ContainerID string `json:"cid,omitempty"`
	MediaID     string `json:"mid,omitempty"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Artist      string `json:"artist,omitempty"`
	Album       string `json:"album,omitempty"`
	AlbumID     string `json:"album_id,omitempty"`
	Playable    bool   `json:"playable"`
	ImageURL    string `json:"image_url,omitempty"`
	// Quality: from HEOS payload when present (e.g. Tidal: LOSSLESS, HI_RES, HIGH, NORMAL).
	Quality string `json:"quality,omitempty"`
}

// SearchResult holds search response and items.
type SearchResult struct {
	SourceID   int             `json:"-"`
	Scid       int             `json:"-"`
	Search     string          `json:"-"`
	Returned   int             `json:"-"`
	Count      int             `json:"-"`
	Items      []MediaItem     `json:"items"`
	RawPayload json.RawMessage `json:"-"` // optional: raw payload for debugging
}

// GetMusicSources returns available music sources. refresh forces a refresh from the device.
func (c *Client) GetMusicSources(ctx context.Context, refresh bool) ([]MusicSource, error) {
	if err := c.Connect(ctx); err != nil {
		return nil, err
	}
	params := map[string]string{}
	if refresh {
		params[AttrRefresh] = ValueOn
	}
	resp, _, err := c.Command(ctx, CmdBrowseGetSources, params)
	if err != nil {
		return nil, err
	}
	var list []MusicSource
	if len(resp.Payload) == 0 {
		return list, nil
	}
	// Payload can be array of objects with "available" as string "true"/"false"
	var raw []map[string]any
	if err := json.Unmarshal(resp.Payload, &raw); err != nil {
		return nil, fmt.Errorf("parse get_music_sources payload: %w", err)
	}
	for _, m := range raw {
		sid := toInt(m["sid"])
		av, _ := m["available"].(string)
		list = append(list, MusicSource{
			SID:             sid,
			Name:            getStr(m, "name"),
			Type:            getStr(m, "type"),
			Available:       av == "true",
			ImageURL:        getStr(m, "image_url"),
			ServiceUsername: getStr(m, "service_username"),
		})
	}
	return list, nil
}

// GetSearchCriteria returns search criteria for a source (e.g. Tidal, SoundCloud).
func (c *Client) GetSearchCriteria(ctx context.Context, sourceID int) ([]SearchCriteria, error) {
	if err := c.Connect(ctx); err != nil {
		return nil, err
	}
	resp, _, err := c.Command(ctx, CmdBrowseGetSearchCriteria, map[string]string{
		AttrSourceID: strconv.Itoa(sourceID),
	})
	if err != nil {
		return nil, err
	}
	var raw []map[string]any
	if len(resp.Payload) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(resp.Payload, &raw); err != nil {
		return nil, fmt.Errorf("parse get_search_criteria payload: %w", err)
	}
	var list []SearchCriteria
	for _, m := range raw {
		scid := toInt(m["scid"])
		list = append(list, SearchCriteria{
			Name:     getStr(m, "name"),
			Scid:     scid,
			Wildcard: getStr(m, "wildcard"),
			Cid:      getStr(m, "cid"),
			Playable: getStr(m, "playable"),
		})
	}
	return list, nil
}

// Search runs a search on a source with the given criteria. Returns items (tracks, etc.).
func (c *Client) Search(ctx context.Context, sourceID int, search string, criteriaID int) (*SearchResult, error) {
	if err := c.Connect(ctx); err != nil {
		return nil, err
	}
	if search == "" {
		return nil, errors.New("search query must not be empty")
	}
	if utf8.RuneCountInString(search) > 128 {
		return nil, errors.New("search query must be at most 128 characters")
	}
	params := map[string]string{
		AttrSourceID:         strconv.Itoa(sourceID),
		AttrSearch:           search,
		AttrSearchCriteriaID: strconv.Itoa(criteriaID),
	}
	resp, msgMap, err := c.Command(ctx, CmdBrowseSearch, params)
	if err != nil {
		return nil, err
	}
	out := &SearchResult{
		SourceID:   sourceID,
		Scid:       criteriaID,
		Search:     search,
		RawPayload: resp.Payload,
	}
	if msgMap[AttrReturned] != "" {
		out.Returned, _ = strconv.Atoi(msgMap[AttrReturned])
	}
	if msgMap[AttrCount] != "" {
		out.Count, _ = strconv.Atoi(msgMap[AttrCount])
	}
	if len(resp.Payload) == 0 {
		return out, nil
	}
	var raw []map[string]any
	if err := json.Unmarshal(resp.Payload, &raw); err != nil {
		return nil, fmt.Errorf("parse search payload: %w", err)
	}
	for _, m := range raw {
		playable := getStr(m, "playable") == "yes"
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
		out.Items = append(out.Items, MediaItem{
			SourceID:    sourceID,
			ContainerID: getStr(m, "cid"),
			MediaID:     getStr(m, "mid"),
			Name:        getStr(m, "name"),
			Type:        getStr(m, "type"),
			Artist:      getStr(m, "artist"),
			Album:       getStr(m, "album"),
			AlbumID:     getStr(m, "album_id"),
			Playable:    playable,
			ImageURL:    getStr(m, "image_url"),
			Quality:     quality,
		})
	}
	return out, nil
}

// PlayStream plays a stream/station on a player (source_id, container_id, media_id, or url).
func (c *Client) PlayStream(ctx context.Context, playerID int, sourceID int, containerID, mediaID string) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}
	params := map[string]string{
		AttrPlayerID: strconv.Itoa(playerID),
		AttrSourceID: strconv.Itoa(sourceID),
		AttrMediaID:  mediaID,
	}
	if containerID != "" {
		params[AttrContainerID] = containerID
	}
	_, _, err := c.Command(ctx, CmdBrowsePlayStream, params)
	return err
}

// PlayStreamURL plays a URL on a player.
func (c *Client) PlayStreamURL(ctx context.Context, playerID int, url string) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}
	_, _, err := c.Command(ctx, CmdBrowsePlayStream, map[string]string{
		AttrPlayerID: strconv.Itoa(playerID),
		AttrURL:      url,
	})
	return err
}

// AddToQueue adds a container/track to the player queue. addCriteria: 1 = play now, 2 = play next, 3 = add to end.
func (c *Client) AddToQueue(ctx context.Context, playerID int, sourceID int, containerID string, mediaID string, addCriteria int) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}
	params := map[string]string{
		AttrPlayerID:      strconv.Itoa(playerID),
		AttrSourceID:      strconv.Itoa(sourceID),
		AttrContainerID:   containerID,
		AttrAddCriteriaID: strconv.Itoa(addCriteria),
	}
	if mediaID != "" {
		params[AttrMediaID] = mediaID
	}
	_, _, err := c.Command(ctx, CmdBrowseAddToQueue, params)
	return err
}

func getStr(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

// toInt reads a JSON number or numeric string; anything else is 0.
func toInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case string:
		n, _ := strconv.Atoi(x)
		return n
	default:
		return 0
	}
}
