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
	"os"
	"strconv"
	"strings"

	"github.com/SENERGY-Platform/mgw-wmbus-dc/pkg/logrotate"
	"github.com/SENERGY-Platform/mgw-wmbus-dc/pkg/model"
	nimbusmgw "github.com/SENERGY-Platform/mgw-wmbus-dc/pkg/nimbus_mgw"
	"github.com/SENERGY-Platform/mgw-wmbus-dc/pkg/util"
	"github.com/nxadm/tail"
)

type encryptedExtractor struct {
	msg *model.EncryptedMessage
}

const (
	meter          = "Received telegram from: "
	manufacturer   = "          manufacturer: "
	_type          = "                  type: "
	version        = "                   ver: "
	rssi           = "                  rssi: "
	device         = "                device: "
	driver         = "                driver: "
	telegramPrefix = "telegram=|_"
	telegramSuffix = "|"
)

func (w *WmbusLogForwarder) handleWmbusmetersLogFile() {
	// add log rotation, since not done by wmbusmeters
	w.logRotater.AddFiles(w.cfg.WmbusLogFile)
	filenameParts := strings.Split(w.cfg.WmbusLogFile, string(os.PathSeparator))
	seekio, err := logrotate.NewSeekIO(w.cfg.SeekDir+string(os.PathSeparator)+filenameParts[len(filenameParts)-1], w.ctx, w.wg)
	if err != nil {
		w.cf()
		return
	}

	seekinfo := seekio.Get()
	seekinfo, err = ensureSeekInfoInBounds(w.cfg.WmbusLogFile, seekinfo)
	if err != nil {
		util.Logger.Error("unable to ensureSeekInfoInBounds", "file", w.cfg.WmbusLogFile, "err", err)
		w.cf()
		return
	}

	w.deviceManager.AddIdempotent(&nimbusmgw.Device{
		Id:           w.cfg.NimbusId,
		Name:         w.cfg.NimbusName,
		DeviceTypeId: w.cfg.NimbusDeviceTypeId,
	})

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		t, err := tail.TailFile(w.cfg.WmbusLogFile, tail.Config{
			Follow:        true,
			Logger:        tail.DiscardingLogger,
			ReOpen:        true,
			CompleteLines: true,
			Location:      seekinfo,
		})
		if err != nil {
			util.Logger.Error("Error tailing file", "error", err)
			w.cf()
		}
		defer t.Cleanup()

		encryptedExtractor := encryptedExtractor{}
		for {
			select {
			case line := <-t.Lines:
				seekio.Set(&line.SeekInfo)
				if line == nil {
					continue
				}
				msg := encryptedExtractor.handleLine(line.Text)
				if msg == nil {
					continue
				}
				util.Logger.Debug("Got message", "meter_id", msg.MeterId, "rssi", msg.RSSI)
				err = w.mgwClient.MarshalAndSendEvent(w.cfg.NimbusId, encryptedServiceId, msg)
				if err != nil {
					util.Logger.Error("unable to send event ("+encryptedServiceId+")", "err", err)
					continue
				}

			case <-w.ctx.Done():
				return
			}
		}
	}()
}

func (e *encryptedExtractor) handleLine(line string) *model.EncryptedMessage {
	if e.msg == nil {
		e.msg = &model.EncryptedMessage{}
	}

	if after, ok := strings.CutPrefix(line, meter); ok {
		e.msg.MeterId = after
		return nil
	}
	if after, ok := strings.CutPrefix(line, manufacturer); ok {
		e.msg.Manufacturer = after
		return nil
	}
	if after, ok := strings.CutPrefix(line, _type); ok {
		e.msg.Type = after
		return nil
	}
	if strings.HasPrefix(line, version) {
		e.msg.Version = strings.TrimPrefix(line, version)
		return nil
	}
	if after, ok := strings.CutPrefix(line, rssi); ok {
		rssi := after
		parts := strings.Split(rssi, " ")
		rssinum, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			util.Logger.Warn("unable to parse rssi to float", "string", parts[0])
			return nil
		}
		e.msg.RSSI = rssinum
		if len(parts) > 1 {
			e.msg.RSSIUnit = parts[1]
		}
		return nil
	}
	if after, ok := strings.CutPrefix(line, driver); ok {
		e.msg.Driver = after
		return nil
	}
	if after, ok := strings.CutPrefix(line, device); ok {
		e.msg.Device = after
		return nil
	}
	if after, ok := strings.CutPrefix(line, telegramPrefix); ok {
		tg := after
		parts := strings.Split(tg, telegramSuffix)
		e.msg.Telegram = parts[0]

		// the telegram is always the last line
		p := e.msg
		e.msg = nil
		return p
	}
	util.Logger.Info("ignored line with unhandled prefix", "line", line)
	return nil
}
