## govisor

[![Join the chat at https://gitter.im/gdamore/govisor](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/gdamore/govisor?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

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
