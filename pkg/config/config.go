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

package config

import (
	sb_config_hdl "github.com/SENERGY-Platform/go-service-base/config-hdl"
)

type Config struct {
	LogLevel              string `json:"log_level" env_var:"LOG_LEVEL"`
	WmbusLogFile          string `json:"wmbus_log_file" env_var:"WMBUS_LOG_FILE"`
	WmbusMeterReadingsDir string `json:"wmbus_meter_readings_dir" env_var:"WMBUS_METER_READINGS_DIR"`
	SeekDir               string `json:"seek_dir" env_var:"SEEK_DIR"`
	LogBackupDir          string `json:"log_backup_dir" env_var:"LOG_BACKUP_DIR"`
	MqttConnStr           string `json:"mqtt_conn_str" env_var:"MQTT_CONN_STR"`
	NimbusId              string `json:"nimbus_id" env_var:"NIMBUS_ID"`
	NimbusName            string `json:"nimbus_name" env_var:"NIMBUS_NAME"`
	NimbusDeviceTypeId    string `json:"nimbus_device_type_id" env_var:"NIMBUS_DEVICE_TYPE_ID"`
}

func New(path string) (*Config, error) {
	cfg := Config{
		LogLevel:              "debug",
		WmbusLogFile:          "/logs/wmbusmeters.log",
		WmbusMeterReadingsDir: "/logs/meter_readings",
		LogBackupDir:          "/logs/backups",
		SeekDir:               "/logs/seeks",
		MqttConnStr:           "tcp://localhost:1883",
		NimbusId:              "nimbus",
		NimbusName:            "nimbus",
		NimbusDeviceTypeId:    "urn:infai:ses:device-type:ae92bb03-fa0d-467e-8c4f-1892dd8494de",
	}
	err := sb_config_hdl.Load(&cfg, nil, envTypeParser, nil, path)
	return &cfg, err
}
