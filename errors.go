// Copyright 2015 The Govisor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use file except in compliance with the License.
// You may obtain a copy of the license at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package govisor

import (
	"errors"
)

var (
	ErrNoManager    = errors.New("No manager for service")
	ErrConflict     = errors.New("Conflicting service enabled")
	ErrIsEnabled    = errors.New("Service is enabled")
	ErrNotRunning   = errors.New("Service is not running")
	ErrBadPropType  = errors.New("Bad property type")
	ErrBadPropName  = errors.New("Bad property name")
	ErrBadPropValue = errors.New("Bad property value")
	ErrPropReadOnly = errors.New("Property not changeable")
	ErrRateLimited  = errors.New("Restarting too quickly")
)
