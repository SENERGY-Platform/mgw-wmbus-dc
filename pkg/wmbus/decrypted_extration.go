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
	"encoding/json"
	"os"
	"strings"

	"github.com/SENERGY-Platform/mgw-wmbus-dc/pkg/logrotate"
	nimbusmgw "github.com/SENERGY-Platform/mgw-wmbus-dc/pkg/nimbus_mgw"
	"github.com/SENERGY-Platform/mgw-wmbus-dc/pkg/util"
	"github.com/fsnotify/fsnotify"
	"github.com/nxadm/tail"
)

func (w *WmbusLogForwarder) handleWmbusmetersMeterReadingDirectory() {
	err := os.MkdirAll(w.cfg.WmbusMeterReadingsDir, 0744)
	if err != nil {
		util.Logger.Error("unable to create wmbusmeters meter reading dir", "dir", w.cfg.WmbusMeterReadingsDir, "err", err)
		w.cf()
		return
	}
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		// check all files in the meter readings dir
		dirEntries, err := os.ReadDir(w.cfg.WmbusMeterReadingsDir)
		if err != nil {
			util.Logger.Error("unable to stat wmbusmeters meter reading dir", "dir", w.cfg.WmbusMeterReadingsDir, "err", err)
			w.cf()
			return
		}
		for _, dirEntry := range dirEntries {
			if dirEntry.IsDir() {
				continue
			}
			w.handleWmbusmetersMeterReadingsFile(w.cfg.WmbusMeterReadingsDir + string(os.PathSeparator) + dirEntry.Name())
		}

		// check for newly created files in the meter readings dir
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			util.Logger.Error("unable to watch wmbusmeters meter reading dir", "dir", w.cfg.WmbusMeterReadingsDir, "err", err)
			w.cf()
			return
		}
		defer watcher.Close()
		err = watcher.Add(w.cfg.WmbusMeterReadingsDir)
		for {
			select {
			case event := <-watcher.Events:
				if event.Op.Has(fsnotify.Create) {
					w.handleWmbusmetersMeterReadingsFile(event.Name)
				}
			case <-w.ctx.Done():
				return
			}
		}
	}()
}

func (w *WmbusLogForwarder) handleWmbusmetersMeterReadingsFile(file string) {
	// add log rotation, since not done by wmbusmeters
	w.logRotater.AddFiles(file)
	filenameParts := strings.Split(file, string(os.PathSeparator))
	seekio, err := logrotate.NewSeekIO(w.cfg.SeekDir+string(os.PathSeparator)+filenameParts[len(filenameParts)-1], w.ctx, w.wg)
	if err != nil {
		w.cf()
		return
	}
	seekinfo := seekio.Get()
	seekinfo, err = ensureSeekInfoInBounds(file, seekinfo)
	if err != nil {
		util.Logger.Error("unable to ensureSeekInfoInBounds", "file", file, "err", err)
		w.cf()
		return
	}

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		t, err := tail.TailFile(file, tail.Config{
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

		for {
			select {
			case line := <-t.Lines:
				seekio.Set(&line.SeekInfo)
				if line == nil {
					continue
				}
				j := map[string]any{}
				err = json.Unmarshal([]byte(line.Text), &j)
				if err != nil {
					util.Logger.Error("unable to unmarshal meter reading line", "file", file, "line", line, "err", err)
					continue
				}
				id, ok := j["id"]
				if !ok {
					util.Logger.Error("unable to read meter reading: missing field id", "file", file, "json", j)
					continue
				}
				idStr, ok := id.(string)
				if !ok {
					util.Logger.Error("unable to read meter reading: field id is not string", "file", file, "json", j)
					continue
				}

				name, ok := j["name"]
				if !ok {
					util.Logger.Error("unable to read meter reading: missing field name", "file", file, "json", j)
					continue
				}
				nameStr, ok := name.(string)
				if !ok {
					util.Logger.Error("unable to read meter reading: field name is not string", "file", file, "json", j)
					continue
				}

				util.Logger.Debug("Got decrypted message", "meter_id", idStr, "name", nameStr)
				w.deviceManager.AddIdempotent(&nimbusmgw.Device{
					Id:   idStr,
					Name: nameStr,
				})
				err = w.mgwClient.SendEvent(idStr, decryptedServiceId, []byte(line.Text))
				util.Logger.Error("unable to send event ("+decryptedServiceId+")", "err", err)
				continue
			case <-w.ctx.Done():
				return
			}
		}
	}()
}
