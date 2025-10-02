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
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/SENERGY-Platform/mgw-wmbus-dc/pkg/util"
	"github.com/nxadm/tail"
)

// Enables presistant saving of tail.SeekInfo
type SeekIO struct {
	file *os.File
}

func NewSeekIO(file string, ctx context.Context, wg *sync.WaitGroup) (*SeekIO, error) {
	f, err := os.OpenFile(file, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil && !os.IsExist(err) {
		util.Logger.Error("Error creating file", "file", file, "err", err)
		return nil, err
	}
	wg.Add(1)
	go func() {
		defer f.Close()
		defer wg.Done()
		<-ctx.Done()
	}()
	return &SeekIO{
		file: f,
	}, nil
}

func (s *SeekIO) Get() *tail.SeekInfo {
	data, err := io.ReadAll(s.file)
	if err != nil {
		util.Logger.Error("Error reading seek info", "err", err)
		return nil
	}
	if len(data) == 0 {
		return nil
	}
	var seekInfo tail.SeekInfo
	err = json.Unmarshal(data, &seekInfo)
	if err != nil {
		util.Logger.Error("Error unmarshaling seek info", "err", err)
		return nil
	}
	return &seekInfo
}

// Set stores the provided SeekInfo into the underlying file. The method will marshal
// the SeekInfo to JSON, truncate the file, and write the data at the beginning of the file.
// If any error occurs during marshaling, truncating, or writing, it will be logged and
// the operation will be aborted.
func (s *SeekIO) Set(info *tail.SeekInfo) {
	data, err := json.Marshal(info)
	if err != nil {
		util.Logger.Error("Error marshaling seek info", "err", err)
		return
	}
	err = s.file.Truncate(0)
	if err != nil {
		util.Logger.Error("Error truncating seek file", "err", err)
		return
	}
	_, err = s.file.WriteAt(data, 0)
	if err != nil {
		util.Logger.Error("Error writing seek info", "err", err)
		return
	}
}
