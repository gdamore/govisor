## govisor

package govisor is an framework for managing services.  It supports dependency
graphs of services, and handles starting, stopping, and restarting services
as necessary.  It also deals with failures, and supports self-healing.

This package is very much a work in progress.  I would discourage using it
directly at this point, though I hope to mature it quickly.

TODO
----

* govisord
* http rest API
* client API
* termbox UI
