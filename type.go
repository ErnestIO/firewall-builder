/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"bytes"
	"encoding/json"
	"log"

	"gopkg.in/redis.v3"
)

type FirewallsCreate struct {
	Service              string     `json:"service"`
	Status               string     `json:"status"`
	ErrorCode            string     `json:"error_code"`
	ErrorMessage         string     `json:"error_message"`
	Firewalls            []firewall `json:"firewalls"`
	Networks             []network  `json:"networks"`
	SequentialProcessing bool       `json:"sequential_processing"`
}

func (e *FirewallsCreate) toJSON() string {
	message, _ := json.Marshal(e)
	return string(message)
}

func (e *FirewallsCreate) cacheKey() string {
	return composeCacheKey(e.Service)
}

func (e *FirewallsCreate) Valid() (bool, string) {
	for _, firewall := range e.Firewalls {
		if ok, msg := firewall.Valid(); ok == false {
			return false, msg
		}
	}
	return true, ""
}

type network struct {
	Name  string `json:"name"`
	Range string `json:"range"`
}

type firewall struct {
	Name               string `json:"name"`
	Rules              []rule `json:"rules"`
	ClientName         string `json:"client_name"`
	RouterType         string `json:"router_type"`
	RouterName         string `json:"router_name"`
	RouterIP           string `json:"router_ip"`
	DatacenterName     string `json:"datacenter_name"`
	DatacenterUsername string `json:"datacenter_username"`
	DatacenterPassword string `json:"datacenter_password"`
	DatacenterType     string `json:"datacenter_type"`
	ExternalNetwork    string `json:"external_network"`
	VCloudURL          string `json:"vcloud_url"`
	Created            bool   `json:"created"`
	Status             string `json:"status"`
	ErrorCode          string `json:"error_code"`
	ErrorMessage       string `json:"error_message"`
}

func (f *firewall) fail() {
	f.Status = "errored"
}

func (f *firewall) complete() {
	f.Status = "completed"
}

func (f *firewall) processing() {
	f.Status = "processed"
}

func (f *firewall) errored() bool {
	return f.Status == "errored"
}

func (f *firewall) completed() bool {
	return f.Status == "completed"
}

func (f *firewall) isProcessed() bool {
	return f.Status == "processed"
}

func (f *firewall) isErrored() bool {
	return f.Status == "errored"
}

func (f *firewall) toBeProcessed() bool {
	return f.Status != "processed" && f.Status != "completed" && f.Status != "errored"
}

func (f *firewall) Valid() (bool, string) {
	if f.Name == "" {
		return false, "Firewall name is empty"
	}
	if f.RouterName == "" {
		return false, "Firewall assigned router is empty"
	}
	return true, ""
}

type firewalls struct {
	Collection []firewall
}

type rule struct {
	SourceIP        string `json:"source_ip"`
	SourcePort      string `json:"source_port"`
	DestinationIP   string `json:"destination_ip"`
	DestinationPort string `json:"destination_port"`
	Protocol        string `json:"protocol"`
}

type firewallEvent struct {
	Service            string `json:"service_id"`
	Type               string `json:"type"`
	Name               string `json:"firewall_name"`
	ClientID           string `json:"client_id"`
	ClientName         string `json:"client_name"`
	Datacenter         string `json:"datacenter_id"`
	DatacenterName     string `json:"datacenter_name"`
	DatacenterUsername string `json:"datacenter_username"`
	DatacenterPassword string `json:"datacenter_password"`
	DatacenterType     string `json:"datacenter_type"`
	ExternalNetwork    string `json:"external_network"`
	VCloudURL          string `json:"vcloud_url"`
	Router             string `json:"router_id"`
	RouterType         string `json:"router_type"`
	RouterName         string `json:"router_name"`
	RouterIP           string `json:"router_ip"`
	Created            bool   `json:"created"`
	FirewallID         string `json:"firewall_id"`
	FirewallType       string `json:"firewall_type"`
	Rules              []rule `json:"firewall_rules"`
}

func (e *firewallEvent) load(f firewall, t string, s string) {
	e.Service = s
	e.Type = t
	e.Name = f.Name
	e.ClientName = f.ClientName
	e.DatacenterName = f.DatacenterName
	e.DatacenterUsername = f.DatacenterUsername
	e.DatacenterPassword = f.DatacenterPassword
	e.DatacenterType = f.DatacenterType
	e.ExternalNetwork = f.ExternalNetwork
	e.VCloudURL = f.VCloudURL
	e.RouterType = f.RouterType
	e.RouterName = f.RouterName
	e.RouterIP = f.RouterIP
	e.FirewallType = "vcloud"
	e.Rules = f.Rules
}

func (e *firewallEvent) toJSON() string {
	message, _ := json.Marshal(e)
	return string(message)
}

type Error struct {
	Code    json.Number `json:"code,Number"`
	Message string      `json:"message"`
}

type firewallCreatedEvent struct {
	Type    string `json:"type"`
	Service string `json:"service_id"`
	Name    string `json:"firewall_name"`
	Error   Error  `json:"error"`
}

func (e *firewallCreatedEvent) cacheKey() string {
	return composeCacheKey(e.Service)
}

type successEvent struct {
	Service   string     `json:"service"`
	Status    string     `json:"status"`
	Firewalls []firewall `json:"firewalls"`
}

func (e *successEvent) toJSON() string {
	message, _ := json.Marshal(e)
	return string(message)
}

type errorEvent struct {
	Service   string `json:"service"`
	Status    string `json:"status"`
	Code      string `json:"error_code"`
	Message   string `json:"error_message"`
	Firewalls []firewall
}

func (e *errorEvent) toJSON() string {
	message, _ := json.Marshal(e)
	return string(message)
}

func persistEvent(redisClient *redis.Client, event *FirewallsCreate) {
	if err := redisClient.Set(event.cacheKey(), event.toJSON(), 0).Err(); err != nil {
		log.Println(err)
	}
}

// Redis Client
type Redis struct {
	Addr     string
	Password string
	DB       int64
}

func composeCacheKey(service string) string {
	var key bytes.Buffer
	key.WriteString("GPBFirewalls_")
	key.WriteString(service)

	return key.String()
}
