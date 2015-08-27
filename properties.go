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

// Property names.  Internal names will all start with an underscore.
// Other, provider specific names, may be supplied.  Note that there is
// no provision for property discovery.  Consumers wishing to use a property
// must know the property name and type.
type PropertyName string

const (
	PropLogger      PropertyName = "_Logger"      // Where logs get sent
	PropRestart                  = "_Restart"     // Auto-restart on failure
	PropRateLimit                = "_RateLimit"   // Max starts per period
	PropRatePeriod               = "_RatePeriod"  // Period for RateLimit
	PropName                     = "_Name"        // Service name
	PropDescription              = "_Description" // Service description
	PropDepends                  = "_Depends"     // Dependencies list
	PropConflicts                = "_Conflicts"   // Conflicts list
	PropProvides                 = "_Provides"    // Provides list
	PropNotify                   = "_Notify"      // Notification callback
)
