/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import "runtime"

var index = 0

func main() {
	n := natsClient()
	r := redisClient()

	// Process requests
	processRequest(n, r, "firewalls.create", "firewall.create")
	processRequest(n, r, "firewalls.update", "firewall.update")
	processRequest(n, r, "firewalls.delete", "firewall.delete")

	// Process resulting success
	processResponse(n, r, "firewall.create.done", "firewalls.create.", "firewall.create", "completed")
	processResponse(n, r, "firewall.update.done", "firewalls.update.", "firewall.update", "completed")
	processResponse(n, r, "firewall.delete.done", "firewalls.delete.", "firewall.delete", "completed")

	// Process resulting errors
	processResponse(n, r, "firewall.create.error", "firewalls.create.", "firewall.create", "errored")
	processResponse(n, r, "firewall.update.error", "firewalls.create.", "firewall.update", "errored")
	processResponse(n, r, "firewall.delete.error", "firewalls.delete.", "firewall.delete", "errored")

	runtime.Goexit()
}
