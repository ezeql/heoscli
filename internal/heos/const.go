package heos

// CLI port for HEOS protocol.
const CLIPort = 1255

// Attribute names (HEOS CLI query keys).
const (
	AttrAddCriteriaID     = "aid"
	AttrCommand           = "command"
	AttrContainerID       = "cid"
	AttrCount             = "count"
	AttrEnable            = "enable"
	AttrErrorID           = "eid"
	AttrMessage           = "message"
	AttrName              = "name"
	AttrPassword          = "pw"
	AttrPayload           = "payload"
	AttrPlayerID          = "pid"
	AttrRange             = "range"
	AttrRefresh           = "refresh"
	AttrResult            = "result"
	AttrReturned          = "returned"
	AttrSearch            = "search"
	AttrSearchCriteriaID  = "scid"
	AttrSourceID          = "sid"
	AttrState             = "state"
	AttrStep              = "step"
	AttrURL               = "url"
	AttrUserName          = "un"
	AttrLevel             = "level"
	AttrMediaID           = "mid"
	AttrHeos              = "heos"
	AttrSignedIn          = "signed_in"
	AttrIPAddress         = "ip"
	AttrModel             = "model"
	AttrSerial            = "serial"
	AttrVersion           = "version"
	AttrType              = "type"
	AttrImageURL          = "image_url"
	AttrAlbum             = "album"
	AttrArtist            = "artist"
	AttrDuration          = "duration"
	AttrSong              = "song"
	AttrStation           = "station"
)

// Result value.
const ValueSuccess = "success"

// On/off values.
const ValueOn = "on"
const ValueOff = "off"

// Add to queue criteria (aid).
const AddCriteriaPlayNow = 1

// Music source IDs (HEOS built-in).
const (
	SourceIDSoundCloud = 9
	SourceIDTidal     = 10
)

// Search criteria container prefix for add_to_queue (search results).
const SearchCriteriaTracksPrefix = "SEARCHED_TRACKS-"

// System commands.
const (
	CmdAccountCheck             = "system/check_account"
	CmdSignIn                   = "system/sign_in"
	CmdSignOut                  = "system/sign_out"
	CmdRegisterForChangeEvents  = "system/register_for_change_events"
	CmdHeartBeat                = "system/heart_beat"
	CmdReboot                   = "system/reboot"
)

// Player commands.
const (
	CmdGetPlayers         = "player/get_players"
	CmdGetPlayerInfo      = "player/get_player_info"
	CmdGetPlayState       = "player/get_play_state"
	CmdSetPlayState       = "player/set_play_state"
	CmdGetNowPlayingMedia = "player/get_now_playing_media"
	CmdGetVolume          = "player/get_volume"
	CmdSetVolume          = "player/set_volume"
	CmdVolumeUp           = "player/volume_up"
	CmdVolumeDown         = "player/volume_down"
	CmdGetMute            = "player/get_mute"
	CmdSetMute            = "player/set_mute"
	CmdToggleMute         = "player/toggle_mute"
	CmdPlayNext           = "player/play_next"
	CmdPlayPrevious       = "player/play_previous"
	CmdGetQueue           = "player/get_queue"
	CmdClearQueue         = "player/clear_queue"
)

// Browse commands.
const (
	CmdBrowseGetSources        = "browse/get_music_sources"
	CmdBrowseGetSourceInfo     = "browse/get_source_info"
	CmdBrowseGetSearchCriteria = "browse/get_search_criteria"
	CmdBrowseSearch            = "browse/search"
	CmdBrowseMultiSearch       = "browse/multi_search"
	CmdBrowsePlayStream        = "browse/play_stream"
	CmdBrowseAddToQueue        = "browse/add_to_queue"
	CmdBrowseBrowse            = "browse/browse"
)
