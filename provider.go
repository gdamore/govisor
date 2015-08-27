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

// Provider is what service providers must implement.  Note that except for
// the Name and Dependencies elements, the service manager promises not to
// call these methods concurrently.  That is, implementers need not worry
// about locking.  Applications should not use this interface.
type Provider interface {
	// Name returns the name of the provider.  For example, a
	// provider for SMTP could return return "smtp" here.
	// For services that have variants, an optioanl variant can be
	// returned by appending a colon.  For example, "smtp:sendmail"
	// indicates a variant using Berkeley sendmail, whereas "smtp:qmail"
	// indicates a variant using qmail.  Both of these would satisfy
	// a dependency of "smtp".  Names may include any alpha numeric,
	// or the underscore.  No punctuation (modulo the colon separating
	// primary name and variant) may be used.
	Name() string

	// Description returns what you think.  Should be only 32 characters
	// to avoid UI truncation.
	Description() string

	// Provides returns a list of service names that this provides.
	// The Name is implicitly added, so only additional values need to
	// be listed.
	Provides() []string

	// Depends returns a list of dependencies, that must be satisfied
	// in order for this service to run.  These are names, that can be
	// either fully qualified (such as "smtp:postfix" or just the
	// base name (such as "smtp").
	Depends() []string

	// Conflicts returns a list of incompatible values.  Note that
	// the service itself is excluded when checking, so that one could
	// have a service list conflicts of "smtp", even though it provides
	// "smtp:postfix".  This would ensure that it does not run with any
	// any other service.
	Conflicts() []string

	// Start attempts to start the service.  It blocks until the service
	// is either started successfuly, or has definitively failed.
	Start() error

	// Stop attempts to stop the service.  As with Start, it blocks until
	// the operation is complete.  This is never allowed to fail.
	Stop()

	// Check performs a health check on the service.  This can be just
	// a check that a process is running, or it can include verification
	// by using test protocol packets or queries, or whatever is
	// appropriate.  It runs synchronously as well.  If all is well it
	// returns nil, otherwise it returns an error.  The error message,
	// provided by Error(), should give some clue as to the reason for the
	// failed check.
	Check() error

	// Property returns the value of a property.
	Property(PropertyName) (interface{}, error)

	// SetProperty sets the value of a property.
	SetProperty(PropertyName, interface{}) error
}
