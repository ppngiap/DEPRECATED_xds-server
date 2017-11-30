/*
 * Copyright (C) 2017 "IoT.bzh"
 * Author Sebastien Douheret <sebastien@iot.bzh>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package xdsserver

import (
	"fmt"
	"path/filepath"

	"github.com/iotbzh/xds-server/lib/xsapiv1"
	uuid "github.com/satori/go.uuid"
)

// CrossSDK Hold SDK config
type CrossSDK struct {
	sdk xsapiv1.SDK
}

// NewCrossSDK creates a new instance of Syncthing
func NewCrossSDK(path string) (*CrossSDK, error) {
	// Assume that we have .../<profile>/<version>/<arch>
	s := CrossSDK{
		sdk: xsapiv1.SDK{Path: path},
	}

	s.sdk.Arch = filepath.Base(path)

	d := filepath.Dir(path)
	s.sdk.Version = filepath.Base(d)

	d = filepath.Dir(d)
	s.sdk.Profile = filepath.Base(d)

	// Use V3 to ensure that we get same uuid on restart
	s.sdk.ID = uuid.NewV3(uuid.FromStringOrNil("sdks"), s.sdk.Profile+"_"+s.sdk.Arch+"_"+s.sdk.Version).String()
	s.sdk.Name = s.sdk.Arch + "  (" + s.sdk.Version + ")"

	envFile := filepath.Join(path, "environment-setup*")
	ef, err := filepath.Glob(envFile)
	if err != nil {
		return nil, fmt.Errorf("Cannot retrieve environment setup file: %v", err)
	}
	if len(ef) != 1 {
		return nil, fmt.Errorf("No environment setup file found match %s", envFile)
	}
	s.sdk.EnvFile = ef[0]

	return &s, nil
}

// Get Return SDK definition
func (s *CrossSDK) Get() *xsapiv1.SDK {
	return &s.sdk
}

// GetEnvCmd returns the command used to initialized the environment
func (s *CrossSDK) GetEnvCmd() []string {
	return []string{"source", s.sdk.EnvFile}
}
