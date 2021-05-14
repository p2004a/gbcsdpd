// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package blelistener implements AdvListener which listens for BLE
// advertisements and published them as Advertisement objects via
// channel returned by Advertisements()
//
// Currently this package does it by using org.bluez.Adapter1 interface:
// Starts discovery and listens for changes to ManufacturerData and RSSI
// properties of all org.bluez.Device1 objects under adapter that are
// propagated via org.freedesktop.DBus.Properties.PropertiesChanged signal.
// See https://git.kernel.org/pub/scm/bluetooth/bluez.git/tree/doc for Bluez
// D-Bus API documentation.
//
// In the future this package might switch to org.bluez.AdvertisementMonitor1
// API but it's currently experimental and not available in stable builds.
package blelistener

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/p2004a/gbcsdpd/pkg/backoff"
)

type objectProperties map[string]dbus.Variant

// org.freedesktop.DBus.Properties.PropertiesChanged signal
// see https://dbus.freedesktop.org/doc/dbus-specification.html
type propertiesChangedSignal struct {
	InterfaceName         string
	ChangedProperties     objectProperties
	InvalidatedProperties []string
}

// org.freedesktop.DBus.ObjectManager.InterfacesRemoved signal
// see https://dbus.freedesktop.org/doc/dbus-specification.html
type interfacesRemovedSignal struct {
	ObjectPath dbus.ObjectPath
	Interfaces []string
}

// org.freedesktop.DBus.ObjectManager.InterfacesAdded signal
// see https://dbus.freedesktop.org/doc/dbus-specification.html
type interfacesAddedSignal struct {
	ObjectPath              dbus.ObjectPath
	InterfacesAndProperties map[string]objectProperties
}

// org.bluez.Device1.ManufacturerData property
// see https://git.kernel.org/pub/scm/bluetooth/bluez.git/tree/doc/device-api.txt
type manufacturerDataProperty map[uint16]dbus.Variant

// ManufacturerData represents a map from manufacturer ID to raw manufacturer
// data embeded in a BLE advertisement.
type ManufacturerData map[uint16][]byte

// Advertisement represents a BLE advertisement.
type Advertisement struct {
	Address          net.HardwareAddr
	ManufacturerData ManufacturerData
}

func parseDeviceMAC(v dbus.Variant) (net.HardwareAddr, error) {
	var strAddr string
	err := v.Store(&strAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to store address variant to string: %v", err)
	}
	addr, err := net.ParseMAC(strAddr)
	if err != nil {
		return nil, fmt.Errorf("address didn't contain valid MAC: %v", err)
	}
	return addr, nil
}

func parseManufacturerData(v dbus.Variant) (ManufacturerData, error) {
	var md manufacturerDataProperty
	if err := v.Store(&md); err != nil {
		return nil, fmt.Errorf("given data is not a org.bluez.Device1.ManufacturerData: %v", err)
	}
	res := make(ManufacturerData)
	for k, v := range md {
		var data []byte
		if err := v.Store(&data); err != nil {
			return nil, fmt.Errorf("failed to store bytes: %v", err)
		}
		res[k] = data
	}
	return res, nil
}

func parseAdvertisementFromProperties(props objectProperties) (Advertisement, error) {
	var adv Advertisement

	// Get Address
	addrVariant, ok := props["Address"]
	if !ok {
		return adv, fmt.Errorf("org.bluez.Device1 doesn't have Address property")
	}
	addr, err := parseDeviceMAC(addrVariant)
	if err != nil {
		return adv, nil
	}
	adv.Address = addr

	// Get ManufacturerData
	mdVariant, ok := props["ManufacturerData"]
	if ok {
		md, err := parseManufacturerData(mdVariant)
		if err != nil {
			return adv, fmt.Errorf("failed to parse manufacturer data: %v", err)
		}
		adv.ManufacturerData = md
	} else {
		adv.ManufacturerData = make(ManufacturerData)
	}
	return adv, nil
}

// AdvListener uses DBUS Bluez interface to listen for BLE advertisements and
// returns them via Advertisements() channel. When the Advertisements() channel
// is closed, the Err field contains the error.
type AdvListener struct {
	adapter  dbus.BusObject
	conn     *dbus.Conn
	m        sync.Mutex // Guards advCache and Err
	advCache map[dbus.ObjectPath]Advertisement
	signals  chan *dbus.Signal
	results  chan Advertisement
	Err      error
}

// Advertisements returns a channel that AdvListener publishes advertisements on.
func (l *AdvListener) Advertisements() <-chan Advertisement {
	return l.results
}

func (l *AdvListener) publishAdvertisement(objPath dbus.ObjectPath, adv Advertisement) {
	l.m.Lock()
	defer l.m.Unlock()
	l.advCache[objPath] = adv
	if len(adv.ManufacturerData) > 0 {
		l.results <- adv
	}
}

func (l *AdvListener) setError(err error) {
	l.m.Lock()
	defer l.m.Unlock()
	if l.Err == nil {
		l.Err = err
	}
	if err := l.conn.Close(); err != nil {
		log.Printf("Closing system bus connection failed: %v", err)
	}
}

func (l *AdvListener) handlePropertiesChanged(objPath dbus.ObjectPath, changed *propertiesChangedSignal) error {
	if changed.InterfaceName != "org.bluez.Device1" {
		return nil
	}
	publish := false
	l.m.Lock()
	adv, ok := l.advCache[objPath]
	l.m.Unlock()
	if !ok {
		var props objectProperties
		if err := l.conn.Object("org.bluez", objPath).Call("org.freedesktop.DBus.Properties.GetAll", 0, "org.bluez.Device1").Store(&props); err != nil {
			return fmt.Errorf("failed to get all properties of %s: %v", objPath, err)
		}
		var err error
		adv, err = parseAdvertisementFromProperties(props)
		if err != nil {
			return fmt.Errorf("failed parse device properties into Advertisement: %v", err)
		}
		publish = true
	}

	if v, ok := changed.ChangedProperties["ManufacturerData"]; ok {
		md, err := parseManufacturerData(v)
		if err != nil {
			return fmt.Errorf("failed to parse manufacturer data: %v", err)
		}
		adv.ManufacturerData = md
		publish = true
	}

	if _, ok := changed.ChangedProperties["RSSI"]; ok {
		publish = true
	}

	if publish {
		l.publishAdvertisement(objPath, adv)
	}
	return nil
}

func (l *AdvListener) handleInterfacesAdded(added *interfacesAddedSignal) error {
	deviceProps, ok := added.InterfacesAndProperties["org.bluez.Device1"]
	if !ok {
		return nil
	}
	adv, err := parseAdvertisementFromProperties(deviceProps)
	if err != nil {
		return fmt.Errorf("failed parse device properties into Advertisement: %v", err)
	}
	l.publishAdvertisement(added.ObjectPath, adv)
	return nil
}

func (l *AdvListener) signalHandlerLoop() {
	defer close(l.results)

	if err := l.conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.DBus.Properties"),
		dbus.WithMatchMember("PropertiesChanged"),
		dbus.WithMatchPathNamespace(l.adapter.Path())); err != nil {
		l.setError(fmt.Errorf("failed to add matcher for PropertiesChanged signal: %v", err))
		return
	}
	if err := l.conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.DBus.ObjectManager"),
		dbus.WithMatchMember("InterfacesRemoved")); err != nil {
		l.setError(fmt.Errorf("failed to add matcher for InterfacesRemoved signal: %v", err))
		return
	}
	if err := l.conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.DBus.ObjectManager"),
		dbus.WithMatchMember("InterfacesAdded")); err != nil {
		l.setError(fmt.Errorf("failed to add matcher for InterfacesAdded signal: %v", err))
		return
	}

	for signal := range l.signals {
		switch signal.Name {
		case "org.freedesktop.DBus.Properties.PropertiesChanged":
			var changed propertiesChangedSignal
			if err := dbus.Store(
				signal.Body,
				&changed.InterfaceName,
				&changed.ChangedProperties,
				&changed.InvalidatedProperties); err != nil {
				log.Printf("Failed to parse PropertiesChanged signal: %v", err)
				break
			}
			if err := l.handlePropertiesChanged(signal.Path, &changed); err != nil {
				log.Printf("Error in handling PropertiesChanged: %v", err)
			}
		case "org.freedesktop.DBus.ObjectManager.InterfacesAdded":
			var added interfacesAddedSignal
			if err := dbus.Store(
				signal.Body,
				&added.ObjectPath,
				&added.InterfacesAndProperties); err != nil {
				log.Printf("Failed to parse InterfacesAdded signal: %v", err)
				break
			}
			if err := l.handleInterfacesAdded(&added); err != nil {
				log.Printf("Error in handling InterfacesAdded: %v", err)
			}
		case "org.freedesktop.DBus.ObjectManager.InterfacesRemoved":
			var removed interfacesRemovedSignal
			if err := dbus.Store(
				signal.Body,
				&removed.ObjectPath,
				&removed.Interfaces); err != nil {
				log.Printf("Failed to parse InterfacesRemoved signal: %v", err)
				break
			}
			l.m.Lock()
			delete(l.advCache, removed.ObjectPath)
			l.m.Unlock()
		}
	}
}

// Issues a call to org.bluez.Adapter1.StartDiscovery retrying the
// "Resource Not Read" error that can happen transitively when the Bluetooth
// device is still powering on.
func (l *AdvListener) callAdapterStartDiscoveryWithRetry() error {
	for retryNum := 0; true; retryNum++ {
		time.Sleep(backoff.Exponential(retryNum, time.Second, time.Second*5, 2.0))
		call := l.adapter.Call("org.bluez.Adapter1.StartDiscovery", 0)
		if call.Err == nil {
			return nil
		}
		if call.Err.Error() != "Resource Not Ready" || retryNum > 5 {
			return call.Err
		}
		log.Printf("Failed to StartDiscovery, retrying...")
	}
	panic("unreachable")
}

func (l *AdvListener) startDiscoveryLoop() {
	for {
		if err := l.callAdapterStartDiscoveryWithRetry(); err != nil {
			l.setError(fmt.Errorf("failed to start discovery: %v", err))
			return
		}
		// This block is responsible for making sure that adapter is constantly
		// in the discovering mode. The single StartDiscovery should enable discovering
		// indefinitely in the bluetooth daemon but it's possible that bluetooth deamon
		// itself will be restarted/will crash and stop discovery.
		// Instead of having this loop we could also listen to proper signals and detect
		// changes that way, but this way it seems easier and good enough.
		for discovering := true; discovering; {
			time.Sleep(time.Minute * 4)
			err := l.adapter.StoreProperty("org.bluez.Adapter1.Discovering", &discovering)
			if err != nil {
				l.setError(fmt.Errorf("failed to read discovering status: %v", err))
				return
			}
		}
		// We should clear the map becuase cache might be stale.
		l.m.Lock()
		l.advCache = make(map[dbus.ObjectPath]Advertisement)
		l.m.Unlock()

		log.Printf("Discovering stopped, restating...")
	}
}

// NewAdvListener creates new AdvListener and starts listening.
func NewAdvListener(adapterName string) (*AdvListener, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, fmt.Errorf("failed to conntect to system bus: %v", err)
	}
	adapterPath := dbus.ObjectPath(fmt.Sprintf("/org/bluez/%s", adapterName))

	// This is just convenience to make the user interface of library better.
	// Let's return that adapter doesn't exist as clear error before calling StartDiscovery.
	var managedObjects map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	if err := conn.Object("org.bluez", dbus.ObjectPath("/")).Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).Store(&managedObjects); err != nil {
		return nil, fmt.Errorf("failed to get list of Bluetooth adapters: %v", err)
	}
	if _, ok := managedObjects[adapterPath]; !ok {
		return nil, fmt.Errorf("requested to listen on Bluetooth adapter '%s', but it doesn't exist", adapterName)
	}

	l := &AdvListener{
		adapter:  conn.Object("org.bluez", adapterPath),
		conn:     conn,
		advCache: make(map[dbus.ObjectPath]Advertisement),
		signals:  make(chan *dbus.Signal, 10),
		results:  make(chan Advertisement, 10),
	}
	conn.Signal(l.signals)
	go l.signalHandlerLoop()
	go l.startDiscoveryLoop()
	return l, nil
}
