/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"os"
	"runtime"

	l "github.com/ernestio/builder-library"
)

var s l.Scheduler

func main() {
	s.Setup(os.Getenv("NATS_URI"))

	// Process requests
	s.ProcessRequest("firewalls.create", "firewall.create")
	s.ProcessRequest("firewalls.delete", "firewall.delete")
	s.ProcessRequest("firewalls.update", "firewall.update")

	// Process resulting success
	s.ProcessSuccessResponse("firewall.create.done", "firewall.create", "firewalls.create.done")
	s.ProcessSuccessResponse("firewall.delete.done", "firewall.delete", "firewalls.delete.done")
	s.ProcessSuccessResponse("firewall.update.done", "firewall.update", "firewalls.update.done")

	// Process resulting errors
	s.ProcessFailedResponse("firewall.create.error", "firewalls.create.error")
	s.ProcessFailedResponse("firewall.delete.error", "firewalls.delete.error")
	s.ProcessFailedResponse("firewall.update.error", "firewalls.update.error")

	runtime.Goexit()
}
