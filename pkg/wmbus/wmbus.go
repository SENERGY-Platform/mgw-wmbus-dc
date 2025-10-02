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

package wmbus

import (
	"context"
	"os"

	"sync"

	"github.com/SENERGY-Platform/mgw-dc-lib-go/pkg/mgw"
	"github.com/SENERGY-Platform/mgw-wmbus-dc/pkg/config"
	"github.com/SENERGY-Platform/mgw-wmbus-dc/pkg/logrotate"
	nimbusmgw "github.com/SENERGY-Platform/mgw-wmbus-dc/pkg/nimbus_mgw"
	"github.com/SENERGY-Platform/mgw-wmbus-dc/pkg/util"
	"github.com/nxadm/tail"
)

const (
	decryptedServiceId = "decrypted"
	encryptedServiceId = "encrypted"
)

type WmbusLogForwarder struct {
	cfg           *config.Config
	mgwClient     *mgw.Client[nimbusmgw.Device]
	deviceManager *nimbusmgw.DeviceManager
	logRotater    *logrotate.LogRotator
	ctx           context.Context
	cf            context.CancelFunc
	wg            *sync.WaitGroup
}

func NewLogForwarder(cfg *config.Config, mgwClient *mgw.Client[nimbusmgw.Device], deviceManager *nimbusmgw.DeviceManager, ctx context.Context, cf context.CancelFunc, wg *sync.WaitGroup) {
	logRotater := logrotate.NewLogRotator(ctx, wg, logrotate.LogRotatorConfig{
		BackupDir: cfg.LogBackupDir,
		Backups:   2,
	})
	err := os.MkdirAll(cfg.SeekDir, 0744)
	if err != nil {
		util.Logger.Error("unable to create seek dir", "dir", cfg.SeekDir, "err", err)
		cf()
		return
	}
	w := &WmbusLogForwarder{
		cfg:           cfg,
		mgwClient:     mgwClient,
		deviceManager: deviceManager,
		logRotater:    logRotater,
		ctx:           ctx,
		cf:            cf,
		wg:            wg,
	}
	w.handleWmbusmetersLogFile()
	w.handleWmbusmetersMeterReadingDirectory()
}

// Validates that the seekinfo offset is within the file's size. Returns nil seekinfo if file is smaller than
// the offset, indicating file rotation. This causes reading from the start of the file.
func ensureSeekInfoInBounds(file string, seekinfo *tail.SeekInfo) (*tail.SeekInfo, error) {
	if seekinfo == nil {
		return nil, nil
	}
	info, err := os.Stat(file)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if info.Size() < seekinfo.Offset {
		return nil, nil
	}
	return seekinfo, nil
}
