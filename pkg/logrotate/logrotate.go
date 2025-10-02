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

package logrotate

import (
	"context"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/SENERGY-Platform/mgw-wmbus-dc/pkg/util"
)

type LogRotator struct {
	files     []string
	mux       sync.RWMutex
	backupDir string
}

type LogRotatorConfig struct {
	BackupDir string
	Backups   int
}

func NewLogRotator(ctx context.Context, wg *sync.WaitGroup, config LogRotatorConfig) *LogRotator {
	r := &LogRotator{
		files: []string{},
		mux:   sync.RWMutex{},
	}

	ticker := time.NewTicker(24 * time.Hour)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.mux.RLock()
				err := os.MkdirAll(config.BackupDir, 0744)
				if err != nil {
					util.Logger.Error("unable to create backup dir", "dir", config.BackupDir, "err", err)
					r.mux.RUnlock()
					continue
				}
				for _, f := range r.files {
					info, err := os.Stat(f)
					if err != nil {
						util.Logger.Warn("unable to state file", "file", f, "err", err)
						r.mux.RUnlock()
						continue
					}
					filename := info.Name()
					for i := config.Backups + 1; i > 1; i-- {
						err := os.Rename(config.BackupDir+string(os.PathSeparator)+filename+"."+strconv.Itoa(i-1), config.BackupDir+string(os.PathSeparator)+filename+"."+strconv.Itoa(i))
						if err != nil && !os.IsNotExist(err) {
							util.Logger.Warn("unable to rotate file", "file", filename, "err", err)
						}
					}
					err = copyFileContents(f, config.BackupDir+string(os.PathSeparator)+filename+".1")
					if err != nil {
						util.Logger.Warn("unable to backup file", "file", filename, "err", err)
						r.mux.RUnlock()
						continue
					}
					err = os.Truncate(f, 0)
					if err != nil && !os.IsNotExist(err) {
						util.Logger.Warn("unable to truncate file", "file", filename, "err", err)
					}
				}
				r.mux.RUnlock()
			}
		}
	}()

	return r
}

func (l *LogRotator) AddFiles(files ...string) {
	l.mux.Lock()
	defer l.mux.Unlock()
	l.files = append(l.files, files...)
}

func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}
