#!/bin/sh

# Copyright 2016 The Govisor Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use file except in compliance with the License.
# You may obtain a copy of the license at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This test program exists to support testing the process module.
# Its unlikely to be useful for anything else.  See process_test.go
# for usage.  Note that it requires a POSIX shell.  Bash is known to
# work.

error() {
	echo "$*" 1>&2
}

case "$1" in
[0-9]*)
	echo "Sleeping $1 secs (stdout)"
	error "Sleep $1 secs (stderr)"
	sleep $1
	;;
"")
	echo "Sleeping an hour (stdout)"
	error "Sleep an hour (stderr)"
	sleep 3600
	;;

checkwd)
	pwd=`pwd`
	exp="$2"
	if [ "$pwd" != "$exp" ]
	then
		error "CWD of $pwd incorrect, expected $exp"
		exit 1
	fi
	echo "CWD is $pwd"
	exit 0
	;;

fail)
	error "Injected failure"
	exit 2
	;;

exit)
	echo "Clean failure"
	exit 0
	;;
*)
	echo "Unknown argument"
	exit 1
	;;
esac
