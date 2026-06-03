package systray

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/stats"
	"fmt"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	serviceName               = "org.openlinkhub.OpenLinkHub"
	statusPath                = dbus.ObjectPath("/StatusNotifierItem")
	menuPath                  = dbus.ObjectPath("/Menu")
	menuItems                 = map[int32]MenuLayout{}
	menuOrder                 []int32
	menuRevision              uint32 = 1
	menuMutex                 sync.Mutex
	conn                      *dbus.Conn
	dbusMenu                  = "com.canonical.dbusmenu"
	dbusMenuLayoutUpdate      = "com.canonical.dbusmenu.LayoutUpdated"
	dbusIntrospectable        = "org.freedesktop.DBus.Introspectable"
	dbusStatusNotifierItem    = "org.kde.StatusNotifierItem"
	dbusProperties            = "org.freedesktop.DBus.Properties"
	dbusStatusNotifierWatcher = "org.kde.StatusNotifierWatcher"
)

// Standard SNI props
var props = map[string]dbus.Variant{
	"Category":   dbus.MakeVariant("ApplicationStatus"),
	"Id":         dbus.MakeVariant("openlinkhub"),
	"Title":      dbus.MakeVariant("OpenLinkHub"),
	"Status":     dbus.MakeVariant("Active"),
	"IconName":   dbus.MakeVariant("cpu"),
	"ToolTip":    createTooltip(),
	"Menu":       dbus.MakeVariant(menuPath),
	"ItemIsMenu": dbus.MakeVariant(true),
}

type Status struct{}
type MenuLayout struct {
	ID       int32                   // i
	Props    map[string]dbus.Variant // a{sv}
	Children []dbus.Variant          // av
}
type MenuServer struct{}

// Activate exported method from spec
func (s *Status) Activate(x, y int32) *dbus.Error { return nil }

// ContextMenu exported method from spec
func (s *Status) ContextMenu(x, y int32) *dbus.Error { return nil }

// Get exported method from spec
func (s *Status) Get(iface, name string) (dbus.Variant, *dbus.Error) {
	v, ok := props[name]
	if !ok {
		return dbus.Variant{}, dbus.MakeFailedError(fmt.Errorf("no such property %q", name))
	}
	return v, nil
}

// GetAll exported method from spec
func (s *Status) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) { return props, nil }

// Set exported method from spec
func (s *Status) Set(iface, name string, v dbus.Variant) *dbus.Error {
	return dbus.MakeFailedError(fmt.Errorf("read-only"))
}

// Version exported method from spec
func (m *MenuServer) Version() (uint32, *dbus.Error) { return 1, nil }

// Status exported method from spec
func (m *MenuServer) Status() (string, *dbus.Error) { return "normal", nil }

// AboutToShow exported method from spec
func (m *MenuServer) AboutToShow(parentId int32) (bool, *dbus.Error) { return true, nil }

// GetLayout exported method from spec
func (m *MenuServer) GetLayout(parentId, recursionDepth int32, propNames []string) (uint32, MenuLayout, *dbus.Error) {
	menuMutex.Lock()
	defer menuMutex.Unlock()

	var children []dbus.Variant
	for _, id := range menuOrder {
		children = append(children, dbus.MakeVariant(menuItems[id]))
	}
	layout := MenuLayout{ID: 0, Props: map[string]dbus.Variant{}, Children: children}
	return menuRevision, layout, nil
}

// Event exported method from spec
func (m *MenuServer) Event(id int32, eventId string, data dbus.Variant, timestamp uint32) *dbus.Error {
	switch id {
	case 101: // Open Dashboard
		url := fmt.Sprintf("http://%s:%d", config.GetConfig().ListenAddress, config.GetConfig().ListenPort)
		err := openBrowser(url)
		if err != nil {
			fmt.Println("Failed to open browser:", err)
		}
	case 103: // Exit
		os.Exit(0)
	}
	return nil
}

// openBrowser will try to open Dashboard
func openBrowser(url string) error {
	return exec.Command("xdg-open", url).Start()
}

// emitMenuUpdate will send dbus message to update menu
func emitMenuUpdate() {
	if conn != nil {
		err := conn.Emit(menuPath, dbusMenuLayoutUpdate, menuRevision, int32(0))
		if err != nil {
			log.Println("Failed to emit menu update:", err)
		}
	}
}

// addMenuItem will create new menu data structure
func addMenuItem(id int32, props map[string]dbus.Variant) {
	menuMutex.Lock()
	defer menuMutex.Unlock()

	if _, exists := menuItems[id]; !exists {
		menuOrder = append(menuOrder, id)
	}
	menuItems[id] = MenuLayout{ID: id, Props: props}
	menuRevision++
}

// SyncBatteryToMenu will sync battery data to menu
func SyncBatteryToMenu(battery map[string]stats.BatteryStats) {
	// Remove old dynamic battery items first
	clearBatteryItems()

	menuMutex.Lock()
	if len(battery) > 0 {
		menuItems[1].Props["visible"] = dbus.MakeVariant(true)
		menuItems[2].Props["visible"] = dbus.MakeVariant(true)
	} else {
		menuItems[1].Props["visible"] = dbus.MakeVariant(false)
		menuItems[2].Props["visible"] = dbus.MakeVariant(false)
	}
	menuRevision++
	menuMutex.Unlock()

	index := int32(1000)
	for _, info := range battery {
		label := fmt.Sprintf("[%d %%] %s", info.Level, info.Device)
		icon := iconForType(int(info.DeviceType))

		addAfterHeader(index, map[string]dbus.Variant{
			"label":     dbus.MakeVariant(label),
			"icon-name": dbus.MakeVariant(icon),
			"type":      dbus.MakeVariant("normal"),
		})
		index++
	}
	emitMenuUpdate()
}

// addAfterHeader will add a new menu item under defined header. For now after second place (2).
func addAfterHeader(id int32, props map[string]dbus.Variant) {
	menuMutex.Lock()
	defer menuMutex.Unlock()
	menuItems[id] = MenuLayout{ID: id, Props: props}
	insertAfter := int32(2)
	pos := 0
	for i, v := range menuOrder {
		if v == insertAfter {
			pos = i + 1
			break
		}
	}
	menuOrder = append(menuOrder[:pos], append([]int32{id}, menuOrder[pos:]...)...)
	menuRevision++
}

// iconForType will return gtk icon per device type
func iconForType(deviceType int) string {
	switch deviceType {
	case 2:
		return "audio-headset"
	case 1:
		return "input-mouse"
	case 0:
		return "input-keyboard"
	default:
		return "battery-good"
	}
}

// clearBatteryItems wipes menu
func clearBatteryItems() {
	menuMutex.Lock()
	defer menuMutex.Unlock()

	var newOrder []int32
	for _, id := range menuOrder {
		if id < 1000 {
			newOrder = append(newOrder, id)
		} else {
			delete(menuItems, id)
		}
	}
	menuOrder = newOrder
	menuRevision++
}

func Init(ready chan struct{}) {
	de := os.Getenv("XDG_CURRENT_DESKTOP")
	if strings.Contains(strings.ToLower(de), "cinnamon") {
		logger.Log(logger.Fields{}).Warn("Cinnamon is not supported for systray. Due to incomplete support for modern tray menus (StatusNotifierItem), this application cannot run reliably on Cinnamon.")
		close(ready)
		return
	}

	var err error
	conn, err = dbus.ConnectSessionBus()
	if err != nil {
		logger.Log(logger.Fields{"error": err}).Warn("Failed to connect to session bus for systray")
		close(ready)
		return
	}
	defer func(conn *dbus.Conn) {
		err = conn.Close()
		if err != nil {
			logger.Log(logger.Fields{"error": err}).Warn("Failed to close session bus")
		}
	}(conn)

	resp, err := conn.RequestName(serviceName, dbus.NameFlagDoNotQueue)
	if err != nil || resp != dbus.RequestNameReplyPrimaryOwner {
		logger.Log(logger.Fields{"error": err, "resp": resp}).Warn("Systray RequestName failed")
		close(ready)
		return
	}

	// Status
	props["IconName"] = dbus.MakeVariant(config.GetConfig().ConfigPath + "/static/img/512.png")
	status := &Status{}
	err = conn.Export(status, statusPath, dbusStatusNotifierItem)
	if err != nil {
		fmt.Println("org.kde.StatusNotifierItem failed to export status", err)
		return
	}

	err = conn.Export(status, statusPath, dbusProperties)
	if err != nil {
		fmt.Println("org.freedesktop.DBus.Properties failed to export status", err)
		return
	}

	err = conn.Export(introspect.NewIntrospectable(&introspect.Node{
		Name: string(statusPath),
		Interfaces: []introspect.Interface{
			{
				Name: dbusStatusNotifierItem,
				Properties: []introspect.Property{
					{Name: "Category", Type: "s", Access: "read"},
					{Name: "Id", Type: "s", Access: "read"},
					{Name: "Title", Type: "s", Access: "read"},
					{Name: "Status", Type: "s", Access: "read"},
					{Name: "IconName", Type: "s", Access: "read"},
					{Name: "ToolTip", Type: "(sa{sv}sas)", Access: "read"},
					{Name: "Menu", Type: "o", Access: "read"},
					{Name: "ItemIsMenu", Type: "b", Access: "read"},
				},
				Methods: []introspect.Method{
					{
						Name: "Activate",
						Args: []introspect.Arg{
							{Name: "x", Type: "i", Direction: "in"},
							{Name: "y", Type: "i", Direction: "in"},
						},
					},
					{
						Name: "ContextMenu",
						Args: []introspect.Arg{
							{Name: "x", Type: "i", Direction: "in"},
							{Name: "y", Type: "i", Direction: "in"},
						},
					},
				},
			},
			{
				Name:    dbusProperties,
				Methods: introspect.Methods(introspect.IntrospectData),
			},
		},
	}), statusPath, dbusIntrospectable)
	if err != nil {
		fmt.Println("org.freedesktop.DBus.Introspectable failed to export status", err)
		return
	}

	// Menu
	menu := &MenuServer{}
	err = conn.Export(menu, menuPath, dbusMenu)
	if err != nil {
		fmt.Println("Failed to export menu", err)
		return
	}

	err = conn.Export(menu, menuPath, dbusProperties)
	if err != nil {
		fmt.Println("org.freedesktop.DBus.Properties failed to export menu", err)
		return
	}

	err = conn.Export(introspect.NewIntrospectable(&introspect.Node{
		Name: string(menuPath),
		Interfaces: []introspect.Interface{
			{
				Name: dbusMenu,
				Methods: []introspect.Method{
					{
						Name: "Version",
						Args: []introspect.Arg{
							{Name: "version", Type: "u", Direction: "out"},
						},
					},
					{
						Name: "Status",
						Args: []introspect.Arg{
							{Name: "status", Type: "s", Direction: "out"},
						},
					},
					{
						Name: "AboutToShow",
						Args: []introspect.Arg{
							{Name: "parentId", Type: "i", Direction: "in"},
							{Name: "needsUpdate", Type: "b", Direction: "out"},
						},
					},
					{
						Name: "GetLayout",
						Args: []introspect.Arg{
							{Name: "parentId", Type: "i", Direction: "in"},
							{Name: "recursionDepth", Type: "i", Direction: "in"},
							{Name: "propertyNames", Type: "as", Direction: "in"},
							{Name: "revision", Type: "u", Direction: "out"},
							{Name: "layout", Type: "(ia{sv}av)", Direction: "out"},
						},
						Annotations: []introspect.Annotation{
							{
								Name:  "org.qtproject.QtDBus.QtTypeName.Out1",
								Value: "DBusMenuLayoutItem",
							},
						},
					},
					{
						Name: "Event", Args: []introspect.Arg{
							{Name: "id", Type: "i", Direction: "in"},
							{Name: "eventId", Type: "s", Direction: "in"},
							{Name: "data", Type: "v", Direction: "in"},
							{Name: "timestamp", Type: "u", Direction: "in"},
						},
					},
				},
				Signals: []introspect.Signal{
					{
						Name: "LayoutUpdated",
						Args: []introspect.Arg{
							{Name: "revision", Type: "u", Direction: "out"},
							{Name: "parentId", Type: "i", Direction: "out"},
						},
					},
				},
			},
			{
				Name:    dbusProperties,
				Methods: introspect.Methods(introspect.IntrospectData),
			},
		},
	}), menuPath, dbusIntrospectable)
	if err != nil {
		fmt.Println("org.freedesktop.DBus.Introspectable failed to export menu", err)
		return
	}

	// Send it
	conn.Object(dbusStatusNotifierWatcher, "/StatusNotifierWatcher").Call("org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem", 0, serviceName)
	err = conn.Emit(menuPath, dbusMenuLayoutUpdate, uint32(1), int32(0))
	if err != nil {
		fmt.Println("com.canonical.dbusmenu.LayoutUpdated failed:", err)
		return
	}

	// Static items
	addMenuItem(1, map[string]dbus.Variant{
		"label":     dbus.MakeVariant("Battery Status"),
		"enabled":   dbus.MakeVariant(false),
		"visible":   dbus.MakeVariant(false),
		"type":      dbus.MakeVariant("normal"),
		"icon-name": dbus.MakeVariant("battery-good"),
	})
	addMenuItem(2, map[string]dbus.Variant{
		"type":    dbus.MakeVariant("separator"),
		"visible": dbus.MakeVariant(false),
	})

	addMenuItem(100, map[string]dbus.Variant{
		"type": dbus.MakeVariant("separator"),
	})

	addMenuItem(101, map[string]dbus.Variant{
		"label":     dbus.MakeVariant("Open Dashboard"),
		"icon-name": dbus.MakeVariant("applications-internet"),
	})

	addMenuItem(102, map[string]dbus.Variant{
		"type": dbus.MakeVariant("separator"),
	})

	addMenuItem(103, map[string]dbus.Variant{
		"label":     dbus.MakeVariant("Quit"),
		"icon-name": dbus.MakeVariant("application-exit"),
	})
	emitMenuUpdate()

	close(ready) // We good
	select {}
}

// createTooltip will create standard tooltip
func createTooltip() dbus.Variant {
	tooltip := struct {
		Title string
		Icons []interface{}
		Text  string
	}{
		Title: "OpenLinkHub",
		Icons: []interface{}{},
		Text:  "OpenLinkHub",
	}
	return dbus.MakeVariant(tooltip)
}

func InitTray() {
	if !config.GetConfig().EnableSystemTray {
		return
	}

	ready := make(chan struct{})
	go func() {
		Init(ready)
	}()

	<-ready // Wait for systray to be ready

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		// Initial sync
		SyncBatteryToMenu(stats.GetBatteryStats())
		for range ticker.C {
			SyncBatteryToMenu(stats.GetBatteryStats())
		}
	}()
}
