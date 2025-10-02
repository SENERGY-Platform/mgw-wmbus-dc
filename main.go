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

package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	srv_info_hdl "github.com/SENERGY-Platform/go-service-base/srv-info-hdl"
	sb_util "github.com/SENERGY-Platform/go-service-base/util"
	"github.com/SENERGY-Platform/mgw-dc-lib-go/pkg/configuration"
	"github.com/SENERGY-Platform/mgw-dc-lib-go/pkg/mgw"
	"github.com/SENERGY-Platform/mgw-wmbus-dc/pkg/config"
	nimbusmgw "github.com/SENERGY-Platform/mgw-wmbus-dc/pkg/nimbus_mgw"
	"github.com/SENERGY-Platform/mgw-wmbus-dc/pkg/util"
	"github.com/SENERGY-Platform/mgw-wmbus-dc/pkg/wmbus"
)

var version = "0.0.1"

func main() {
	ec := 0
	defer func() {
		os.Exit(ec)
	}()

	srvInfoHdl := srv_info_hdl.New("github.com/SENERGY-Platform/mgw-wmbus-dc", version)

	config.ParseFlags()

	cfg, err := config.New(config.ConfPath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		ec = 1
		return
	}

	util.InitStructLogger(cfg.LogLevel)

	util.Logger.Info(srvInfoHdl.Name(), "version", srvInfoHdl.Version())
	util.Logger.Info("config: " + sb_util.ToJsonStr(cfg))

	ctx, cf := context.WithCancel(context.Background())

	go func() {
		util.Wait(ctx, util.Logger, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		cf()
	}()

	wg := &sync.WaitGroup{}

	var dm *nimbusmgw.DeviceManager

	mgwClient, err := mgw.New[nimbusmgw.Device](configuration.Config{
		ConnectorId:   "mgw-wmbus-dc",
		MgwMqttBroker: cfg.MqttConnStr,
	}, ctx, wg, func() {
		for dm == nil {
			time.Sleep(time.Second)
		}
		dm.Refresh()
	})
	if err != nil {
		util.Logger.Error("unable to create mgw client", "err", err)
		cf()
		return
	}
	dm = nimbusmgw.NewDeviceManager(mgwClient)
	wmbus.NewLogForwarder(cfg, mgwClient, dm, ctx, cf, wg)

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()

	}()

	wg.Wait()
}
