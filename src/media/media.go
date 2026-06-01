package media

// Package: media
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"errors"
	"github.com/godbus/dbus/v5"
	"sort"
	"strings"
	"sync"
)

const (
	dbusMprisPath           = dbus.ObjectPath("/org/mpris/MediaPlayer2")
	dbusPlayerInterface     = "org.mpris.MediaPlayer2.Player"
	dbusPropertiesInterface = "org.freedesktop.DBus.Properties"
	dbusInterface           = "org.freedesktop.DBus"
	dbusObjectPath          = dbus.ObjectPath("/org/freedesktop/DBus")
	dbusMprisPrefix         = "org.mpris.MediaPlayer2."
)

var (
	dbusClient *Client
	mutex      sync.RWMutex
)

type NowPlaying struct {
	Playing        bool     `json:"playing"`
	Service        string   `json:"service"`
	PlaybackStatus string   `json:"playbackStatus"`
	Title          string   `json:"title"`
	Artists        []string `json:"artists"`
	Album          string   `json:"album,omitempty"`
	LengthUs       int64    `json:"length-us"`
	PositionUs     int64    `json:"position-us"`
	Length         float64  `json:"length"`
	Position       float64  `json:"position"`
	TrackID        string   `json:"track-id"`
}

type Client struct {
	mu     sync.RWMutex
	conn   *dbus.Conn
	closed bool
}

// Init will initialize a shared mpris media monitor.
func Init() {
	if config.IsSystemService() {
		logger.Log(logger.Fields{}).Error("Media Client is not available while service is running in system context")
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	if dbusClient != nil {
		logger.Log(logger.Fields{}).Warn("dbus client already initialized")
		return
	}

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Failed to connect to user session bus")
		return
	}
	dbusClient = &Client{conn: conn}
}

// Stop closes the shared MPRIS media monitor connection.
func Stop() {
	mutex.Lock()
	defer mutex.Unlock()

	if dbusClient == nil {
		logger.Log(logger.Fields{}).Warn("dbus client already stopped")
		return
	}

	err := dbusClient.Close()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Error("Failed to close user session bus")
		return
	}
	dbusClient = nil
}

// GetCurrentlyPlaying will return current playing media or error
func GetCurrentlyPlaying() (NowPlaying, error) {
	mutex.RLock()
	defer mutex.RUnlock()

	if dbusClient == nil {
		return NowPlaying{Playing: false}, errors.New("media monitor is not initialized")
	}

	return dbusClient.getCurrentlyPlaying()
}

// Close will close user dbus session and client
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	if c.conn == nil {
		return nil
	}

	return c.conn.Close()
}

// getCurrentlyPlaying will return currently playing media or error
func (c *Client) getCurrentlyPlaying() (NowPlaying, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed || c.conn == nil {
		return NowPlaying{Playing: false}, errors.New("dbus client is closed")
	}

	return getCurrentlyPlaying(c.conn)
}

// getCurrentlyPlaying will return current playing media or error
func getCurrentlyPlaying(conn *dbus.Conn) (NowPlaying, error) {
	players, err := listPlayers(conn)
	if err != nil {
		return NowPlaying{Playing: false}, err
	}

	for _, service := range players {
		np, err := getNowPlaying(conn, service)
		if err != nil {
			continue
		}

		if np.Playing {
			return np, nil
		}
	}
	return NowPlaying{Playing: false}, nil
}

// listPlayers will return list of players
func listPlayers(conn *dbus.Conn) ([]string, error) {
	obj := conn.Object(dbusInterface, dbusObjectPath)

	var names []string
	if err := obj.Call(dbusInterface+".ListNames", 0).Store(&names); err != nil {
		return nil, err
	}

	players := make([]string, 0)

	for _, name := range names {
		if strings.HasPrefix(name, dbusMprisPrefix) {
			players = append(players, name)
		}
	}

	sort.Strings(players)

	return players, nil
}

// getNowPlaying will return currently playing media
func getNowPlaying(conn *dbus.Conn, service string) (NowPlaying, error) {
	obj := conn.Object(service, dbusMprisPath)

	props, err := getAllProperties(obj, dbusPlayerInterface)
	if err != nil {
		return NowPlaying{}, err
	}

	np := NowPlaying{
		Service:        service,
		PlaybackStatus: getPropString(props, "PlaybackStatus"),
		PositionUs:     getPropInt64(props, "Position"),
	}
	np.Position = microsecondsToSeconds(np.PositionUs)
	np.Playing = np.PlaybackStatus == "Playing"

	meta := getMetadata(props)
	np.Title = getMetaString(meta, "xesam:title")
	np.Artists = getMetaStringSlice(meta, "xesam:artist")
	np.Album = getMetaString(meta, "xesam:album")
	np.LengthUs = getMetaInt64(meta, "mpris:length")
	np.Length = microsecondsToSeconds(np.LengthUs)
	np.TrackID = getTrackID(meta)

	return np, nil
}

// getAllProperties will return dbus props based on interface key
func getAllProperties(obj dbus.BusObject, interfaceKey string) (map[string]dbus.Variant, error) {
	var props map[string]dbus.Variant

	err := obj.Call(dbusPropertiesInterface+".GetAll", 0, interfaceKey).Store(&props)
	if err != nil {
		return nil, err
	}

	return props, nil
}

// getMetadata will return metadata from dbus props
func getMetadata(props map[string]dbus.Variant) map[string]dbus.Variant {
	v, ok := props["Metadata"]
	if !ok {
		return nil
	}

	meta, ok := v.Value().(map[string]dbus.Variant)
	if !ok {
		return nil
	}

	return meta
}

// getPropString will return string from dbus props based on key
func getPropString(props map[string]dbus.Variant, key string) string {
	if props == nil {
		return ""
	}

	return dbusVarString(props[key])
}

// getPropInt64 will return int64 from dbus props based on key
func getPropInt64(props map[string]dbus.Variant, key string) int64 {
	if props == nil {
		return 0
	}

	return dbusVarInt64(props[key])
}

// getMetaString will return meta string from dbus meta based on key
func getMetaString(meta map[string]dbus.Variant, key string) string {
	if meta == nil {
		return ""
	}

	return dbusVarString(meta[key])
}

// getMetaInt64 will return meta int64 from dbus meta based on key
func getMetaInt64(meta map[string]dbus.Variant, key string) int64 {
	if meta == nil {
		return 0
	}

	return dbusVarInt64(meta[key])
}

// getPropInt64 will return meta string slice from dbus meta based on key
func getMetaStringSlice(meta map[string]dbus.Variant, key string) []string {
	if meta == nil {
		return nil
	}

	v, ok := meta[key]
	if !ok {
		return nil
	}

	switch x := v.Value().(type) {
	case []string:
		return x

	case []interface{}:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, valid := item.(string); valid {
				out = append(out, s)
			}
		}
		return out

	default:
		return nil
	}
}

// getTrackID will return trackId as string from dbus meta
func getTrackID(meta map[string]dbus.Variant) string {
	if meta == nil {
		return ""
	}

	v, ok := meta["mpris:trackid"]
	if !ok {
		return ""
	}

	switch x := v.Value().(type) {
	case dbus.ObjectPath:
		return string(x)
	case string:
		return x
	default:
		return ""
	}
}

// dbusVarString will convert dbus variant to string
func dbusVarString(v dbus.Variant) string {
	if v.Signature().Empty() {
		return ""
	}

	s, _ := v.Value().(string)
	return s
}

// dbusVarInt64 will convert dbus variant to int64
func dbusVarInt64(v dbus.Variant) int64 {
	if v.Signature().Empty() {
		return 0
	}

	switch x := v.Value().(type) {
	case int64:
		return x
	case int32:
		return int64(x)
	case int16:
		return int64(x)
	case int:
		return int64(x)
	case uint64:
		return int64(x)
	case uint32:
		return int64(x)
	case uint16:
		return int64(x)
	case uint:
		return int64(x)
	default:
		return 0
	}
}

// microsecondsToSeconds will convert micro seconds to seconds
func microsecondsToSeconds(us int64) float64 {
	if us <= 0 {
		return 0
	}

	return float64(us) / 1_000_000.0
}
