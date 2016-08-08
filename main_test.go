/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats"
	"gopkg.in/redis.v3"
)

var setup = false
var n *nats.Conn
var r *redis.Client

func wait(ch chan bool) error {
	return waitTime(ch, 500*time.Millisecond)
}

func waitTime(ch chan bool, timeout time.Duration) error {
	select {
	case <-ch:
		return nil
	case <-time.After(timeout):
	}
	return errors.New("timeout")
}

func testSetup() {
	if setup == false {
		os.Setenv("NATS_URI", "nats://localhost:4222")

		n = natsClient()
		n.Subscribe("config.get.redis", func(msg *nats.Msg) {
			n.Publish(msg.Reply, []byte(`{"DB":0,"addr":"localhost:6379","password":""}`))
		})
		r = redisClient()
		setup = true
	}
}

func TestProvisionAllFirewallsBasic(t *testing.T) {
	testSetup()
	processRequest(n, r, "firewalls.create", "firewall.provision")

	ch := make(chan bool)

	n.Subscribe("firewall.provision", func(msg *nats.Msg) {
		event := &firewallEvent{}
		json.Unmarshal(msg.Data, event)
		if event.Type == "firewall.provision" &&
			event.RouterName == "router" &&
			event.RouterIP == "1.1.1.1" {
			log.Println("Message Received")
			var key bytes.Buffer
			key.WriteString("GPBFirewalls_")
			key.WriteString(event.Service)
			message, _ := r.Get(key.String()).Result()
			stored := &FirewallsCreate{}
			json.Unmarshal([]byte(message), stored)
			if stored.Service != event.Service {
				t.Fatal("Event is not persisted correctly")
			}
			ch <- true
		} else {
			t.Fatal("Invalid received type")
		}
	})

	message := "{\"service\":\"service\", \"firewalls\":[{\"name\":\"name\",\"router\":\"router\",\"router_name\":\"router\",\"client\": \"client\", \"router_ip\": \"1.1.1.1\"}]}"
	n.Publish("firewalls.create", []byte(message))
	time.Sleep(500 * time.Millisecond)

	if e := wait(ch); e != nil {
		t.Fatal("Message not received from nats for subscription")
	}
}

func TestProvisionAllFirewallsWithNetworks(t *testing.T) {
	testSetup()
	processRequest(n, r, "firewalls.create", "firewall.provision")

	ch := make(chan bool)

	n.Subscribe("firewall.provision", func(msg *nats.Msg) {
		event := &firewallEvent{}
		json.Unmarshal(msg.Data, event)
		if event.Type == "firewall.provision" &&
			event.RouterName == "router" {
			if len(event.Rules) == 1 {
				ch <- true
			} else {
				t.Fatal("Invalid number of rules calculated")
			}
		} else {
			t.Fatal("Invalid received type")
		}
	})

	message := "{\"service\":\"service\", \"firewalls\":[{\"name\":\"name\",\"router\":\"router\",\"router_name\":\"router\",\"client\": \"client\", \"rules\":[{\"source_ip\":\"8.8.8.8\",\"source_port\":\"80\",\"destination_ip\":\"8.8.8.8\", \"destination_port\":\"80\",\"protocol\":\"tcp\"}]}], \"networks\": [{\"name\":\"n1\", \"range\": \"1.1.1.1\"},{\"name\":\"n2\", \"range\": \"2.2.2.2\"}, {\"name\":\"n3\", \"range\": \"3.3.3.3\"}]}"
	n.Publish("firewalls.create", []byte(message))
	time.Sleep(500 * time.Millisecond)

	if e := wait(ch); e != nil {
		t.Fatal("Message not received from nats for subscription")
	}
}

func ignoreTestProvisionAllFirewallsSendingTwoFirewalls(t *testing.T) {
	testSetup()
	processRequest(n, r, "firewalls.create", "firewall.provision")

	ch := make(chan bool)
	ch2 := make(chan bool)

	n.Subscribe("firewall.provision", func(msg *nats.Msg) {
		event := &firewallEvent{}
		json.Unmarshal(msg.Data, event)
		if event.Type == "firewall.provision" &&
			event.Name == "name" &&
			event.Service == "service" {
			ch <- true
		} else {
			if event.Type == "firewall.provision" &&
				event.Name == "name2" &&
				event.Service == "service" {
				ch2 <- true
			} else {
				t.Fatal("Message received from nats does not match")
			}
		}
	})

	message := "{\"service\":\"service\", \"firewalls\":[{\"name\":\"name\",\"router\":\"router\",\"client\": \"client\"}, {\"name\":\"name2\",\"router\":\"router2\",\"client2\": \"client2\"}]}"
	n.Publish("firewalls.create", []byte(message))
	time.Sleep(1500 * time.Millisecond)
	if e := wait(ch); e != nil {
		t.Fatal("Message not received from nats for firewall name")
	}
	if e := wait(ch2); e != nil {
		t.Fatal("Message not received from nats for firewall name2")
	}
}

func TestProvisionAllFirewallsWithInvalidMessage(t *testing.T) {
	testSetup()
	processRequest(n, r, "firewalls.create", "firewall.provision")

	ch := make(chan bool)
	ch2 := make(chan bool)

	n.Subscribe("firewall.provision", func(msg *nats.Msg) {
		ch <- true
	})

	n.Subscribe("firewalls.create.error", func(msg *nats.Msg) {
		ch2 <- true
	})

	message := "{\"service\":\"service\", \"firewalls\":[{\"name\":\"\",\"router\":\"router\",\"client\": \"client\"}]}"
	n.Publish("firewalls.create.error", []byte(message))

	if e := wait(ch); e == nil {
		t.Fatal("Produced a firewall.provision message when I shouldn't")
	}
	if e := wait(ch2); e != nil {
		t.Fatal("Should produce a firewalls.create.error message on nats")
	}
}

func TestProvisionAllFirewallsWithDifferentMessageType(t *testing.T) {
	testSetup()
	processRequest(n, r, "firewalls.create", "firewall.provision")

	ch := make(chan bool)

	n.Subscribe("firewall.provision", func(msg *nats.Msg) {
		ch <- true
	})

	message := "{\"service\":\"service\", \"firewalls\":[{\"name\":\"\",\"router_name\":\"router\",\"client_name\": \"client\"}]}"
	n.Publish("non-firewalls-create", []byte(message))

	if e := wait(ch); e == nil {
		t.Fatal("Produced a firewall.provision message when I shouldn't")
	}
}

func TestFirewallCreatedForAMultiRequest(t *testing.T) {
	testSetup()
	processResponse(n, r, "firewall.create.done", "firewalls.create.", "firewall.provision", "completed")

	ch := make(chan bool)
	service := "sss"

	n.Subscribe("firewalls.create.done", func(msg *nats.Msg) {
		t.Fatal("Message received from nats does not match")
	})

	original := "{\"service\":\"sss\", \"firewalls\":[{\"name\":\"name\",\"router_name\":\"router\",\"client_name\": \"client\"},{\"name\":\"name2\",\"router_name\":\"router2\",\"client_name\": \"client2\"}]}"
	if err := r.Set("GPBFirewalls_sss", original, 0).Err(); err != nil {
		log.Println(err)
		t.Fatal("Can't write on redis")
	}
	message := fmt.Sprintf("{\"type\":\"firewall.create.done\",\"service_id\":\"%s\",\"name\":\"name\"}", service)
	n.Publish("firewall.create.done", []byte(message))

	if e := wait(ch); e != nil {
		return
	}
}

func TestFirewallCreated(t *testing.T) {
	testSetup()
	processResponse(n, r, "firewall.create.done", "firewalls.create.", "firewall.provision", "completed")

	ch := make(chan bool)
	service := "sss"

	n.Subscribe("firewalls.create.done", func(msg *nats.Msg) {
		event := &successEvent{}
		json.Unmarshal(msg.Data, event)
		if service == event.Service && event.Status == "completed" && len(event.Firewalls) == 1 {
			ch <- true
		} else {
			t.Fatal("Message received from nats does not match")
		}
	})

	original := "{\"service\":\"sss\", \"firewalls\":[{\"name\":\"name\",\"router_name\":\"router\",\"client_name\": \"client\"}]}"
	if err := r.Set("GPBFirewalls_sss", original, 0).Err(); err != nil {
		log.Println(err)
		t.Fatal("Can't write on redis")
	}
	message := fmt.Sprintf("{\"type\":\"firewall.create.done\",\"service_id\":\"%s\",\"firewall_name\":\"name\"}", service)
	n.Publish("firewall.create.done", []byte(message))

	if e := wait(ch); e != nil {
		t.Fatal("Message not received from nats for subscription")
	}
}

func TestProvisionFirewallError(t *testing.T) {
	testSetup()
	processResponse(n, r, "firewall.create.error", "firewalls.create.", "firewall.provision", "errored")

	ch := make(chan bool)
	service := "sss"

	n.Subscribe("firewalls.create.error", func(msg *nats.Msg) {
		event := &successEvent{}
		json.Unmarshal(msg.Data, event)
		if service == event.Service && event.Status == "error" {
			ch <- true
		} else {
			t.Fatal("Message received from nats does not match")
		}
	})

	original := "{\"service\":\"sss\", \"firewalls\":[{\"name\":\"name\",\"router_name\":\"router\",\"client_name\": \"client\"}]}"
	if err := r.Set("GPBFirewalls_sss", original, 0).Err(); err != nil {
		log.Println(err)
		t.Fatal("Can't write on redis")
	}
	message := fmt.Sprintf("{\"type\":\"firewall.create.error\",\"service_id\":\"%s\",\"firewall_name\":\"name\"}", service)
	n.Publish("firewall.create.error", []byte(message))

	if e := wait(ch); e != nil {
		t.Fatal("Message not received from nats for subscription")
	}
}
