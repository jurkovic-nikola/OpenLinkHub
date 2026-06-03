package systray

import (
	"OpenLinkHub/src/cluster"
	"OpenLinkHub/src/devices"
	"github.com/godbus/dbus/v5"
	"sort"
	"strings"
)

var deviceMap = make(map[int]string)

func isDeviceInCluster(serial string) bool {
	return devices.GetDeviceClusterStatus(serial)
}

func createSubMenuLayout(id int32, label string, items map[int32]string) MenuLayout {
	var children []dbus.Variant

	var childIds []int32
	for k := range items {
		childIds = append(childIds, k)
	}
	sort.Slice(childIds, func(i, j int) bool { return childIds[i] < childIds[j] })

	for _, childId := range childIds {
		childLabel := items[childId]
		childLayout := MenuLayout{
			ID: childId,
			Props: map[string]dbus.Variant{
				"label": dbus.MakeVariant(childLabel),
			},
		}
		children = append(children, dbus.MakeVariant(childLayout))
		menuItems[childId] = childLayout
	}

	layout := MenuLayout{
		ID: id,
		Props: map[string]dbus.Variant{
			"label":            dbus.MakeVariant(label),
			"children-display": dbus.MakeVariant("submenu"),
		},
		Children: children,
	}
	menuItems[id] = layout
	return layout
}

func RefreshDevicesMenu(parentId int32) {
	menuMutex.Lock()
	defer menuMutex.Unlock()

	allDevices := devices.GetDevicesEx()
	var validSerials []string
	for serial := range allDevices {
		if serial != "cluster" {
			validSerials = append(validSerials, serial)
		}
	}
	sort.Strings(validSerials)

	modes := make([]string, len(cluster.Get().RGBModes))
	copy(modes, cluster.Get().RGBModes)
	sort.Strings(modes)

	var devicesChildren []dbus.Variant

	for i, serial := range validSerials {
		deviceMap[i] = serial
		dev := allDevices[serial]

		baseId := int32(1000 + (i * 100))

		childItems := make(map[int32]string)

		inCluster := isDeviceInCluster(serial)
		toggleStr := "[ ] Sync to Global Cluster"
		if inCluster {
			toggleStr = "[✔] Sync to Global Cluster"
		}
		childItems[baseId] = toggleStr

		for j, mode := range modes {
			childItems[baseId+1+int32(j)] = strings.Title(mode)
		}

		devLayout := createSubMenuLayout(baseId+99, dev.Product, childItems)
		devicesChildren = append(devicesChildren, dbus.MakeVariant(devLayout))
	}

	layout := MenuLayout{
		ID: parentId,
		Props: map[string]dbus.Variant{
			"label":            dbus.MakeVariant("Devices"),
			"children-display": dbus.MakeVariant("submenu"),
			"icon-name":        dbus.MakeVariant("preferences-desktop-peripherals"),
		},
		Children: devicesChildren,
	}

	menuItems[parentId] = layout

	found := false
	for _, v := range menuOrder {
		if v == parentId {
			found = true
			break
		}
	}
	if !found {
		menuOrder = append(menuOrder, parentId)
	}

	menuRevision++
	go emitMenuUpdate()
}
