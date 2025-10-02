/*
 * Copyright (c) 2025 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nimbusmgw

import (
	"sync"

	"github.com/SENERGY-Platform/mgw-dc-lib-go/pkg/mgw"
	"github.com/SENERGY-Platform/mgw-wmbus-dc/pkg/util"
)

type Device struct {
	Id           string
	Name         string
	DeviceTypeId string
}

func (d Device) GetInfo() mgw.DeviceInfo {
	return mgw.DeviceInfo{
		Id:         d.Id,
		Name:       d.Name,
		DeviceType: d.DeviceTypeId,
		State:      "online", // actually unknown, but not allowed
	}
}

type DeviceManager struct {
	devices   map[string]*Device
	mux       sync.RWMutex
	mgwClient *mgw.Client[Device]
}

func NewDeviceManager(mgwClient *mgw.Client[Device]) *DeviceManager {
	return &DeviceManager{
		devices:   map[string]*Device{},
		mux:       sync.RWMutex{},
		mgwClient: mgwClient,
	}
}

func (dm *DeviceManager) AddIdempotent(d *Device) error {
	dm.mux.RLock()
	_, ok := dm.devices[d.Id]
	dm.mux.RUnlock()
	if ok {
		return nil
	}
	dm.mux.Lock()
	dm.devices[d.Id] = d
	dm.mux.Unlock()
	return dm.mgwClient.SetDevice(*d)
}

func (dm *DeviceManager) Refresh() {
	dm.mux.RLock()
	defer dm.mux.RUnlock()
	for _, d := range dm.devices {
		err := dm.mgwClient.SetDevice(*d)
		if err != nil {
			util.Logger.Error("unable to set device at mgw", "err", err)
			continue
		}
	}
}
