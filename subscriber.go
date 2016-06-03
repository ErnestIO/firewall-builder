/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats"
	"gopkg.in/redis.v3"
)

func updateFirewall(n *nats.Conn, f firewall, s string, net []network, t string) {
	e := firewallEvent{}
	e.load(f, t, s)
	n.Publish(t, []byte(e.toJSON()))
}

func processRequest(n *nats.Conn, r *redis.Client, subject string, resSubject string) {
	n.Subscribe(subject, func(m *nats.Msg) {
		event := FirewallsCreate{}
		json.Unmarshal(m.Data, &event)

		persistEvent(r, &event)
		if len(event.Firewalls) == 0 || event.Status == "completed" {
			event.Status = "completed"
			event.ErrorCode = ""
			event.ErrorMessage = ""
			n.Publish(subject+".done", []byte(event.toJSON()))
			return
		}
		for _, firewall := range event.Firewalls {
			if ok, msg := firewall.Valid(); ok == false {
				event.Status = "error"
				event.ErrorCode = "0001"
				event.ErrorMessage = msg
				n.Publish(subject+".error", []byte(event.toJSON()))
				return
			}
		}
		sw := false
		for i, firewall := range event.Firewalls {
			if event.Firewalls[i].completed() == false {
				sw = true
				event.Firewalls[i].processing()
				updateFirewall(n, firewall, event.Service, event.Networks, resSubject)
				if true == event.SequentialProcessing {
					break
				}
			}
		}
		if sw == false {
			event.Status = "completed"
			event.ErrorCode = ""
			event.ErrorMessage = ""
			n.Publish(subject+".done", []byte(event.toJSON()))
			return
		}

		persistEvent(r, &event)
	})
}

func processResponse(n *nats.Conn, r *redis.Client, s string, res string, p string, t string) {
	n.Subscribe(s, func(m *nats.Msg) {
		stored, completed := processNext(n, r, s, p, m.Data, t)
		if completed {
			complete(n, stored, res)
		}
	})
}

func complete(n *nats.Conn, stored *FirewallsCreate, subject string) {
	if isErrored(stored) == true {
		stored.Status = "error"
		stored.ErrorCode = "0002"
		stored.ErrorMessage = "Some firewalls could not be successfully processed"
		n.Publish(subject+"error", []byte(stored.toJSON()))
	} else {
		stored.Status = "completed"
		n.Publish(subject+"done", []byte(stored.toJSON()))
	}
}

func isErrored(stored *FirewallsCreate) bool {
	for _, v := range stored.Firewalls {
		if v.isErrored() {
			return true
		}
	}
	return false
}

func processNext(n *nats.Conn, r *redis.Client, subject string, procSubject string, body []byte, status string) (*FirewallsCreate, bool) {
	event := &firewallCreatedEvent{}
	json.Unmarshal(body, event)

	message, err := r.Get(event.cacheKey()).Result()
	if err != nil {
		log.Println(err)
	}
	stored := &FirewallsCreate{}
	json.Unmarshal([]byte(message), stored)
	completed := true
	scheduled := false
	for i := range stored.Firewalls {
		if stored.Firewalls[i].Name == event.Name {
			stored.Firewalls[i].Status = status
			if status == "errored" {
				stored.Firewalls[i].ErrorCode = string(event.Error.Code)
				stored.Firewalls[i].ErrorMessage = event.Error.Message
			}
		}
		if stored.Firewalls[i].completed() == false && stored.Firewalls[i].errored() == false {
			completed = false
		}
		if stored.Firewalls[i].toBeProcessed() && scheduled == false {
			scheduled = true
			completed = false
			stored.Firewalls[i].processing()
			updateFirewall(n, stored.Firewalls[i], event.Service, stored.Networks, procSubject)
		}
	}
	persistEvent(r, stored)

	return stored, completed
}
