// Copyright (c) 2023-2024 Pragmagic Inc. and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

// Node is a graphical model entity
type Node struct {
	Data struct {
		ID         string                 `json:"id"`
		Type       NodeType               `json:"type"`
		Label      string                 `json:"label"`
		Parent     string                 `json:"parent,omitempty"`
		CustomData map[string]interface{} `json:"customData,omitempty"`
	} `json:"data"`
}

// Edge is a graphical model connection
type Edge struct {
	Data struct {
		ID         string                 `json:"id"`
		Type       EdgeType               `json:"type"`
		Label      string                 `json:"label,omitempty"`
		Source     string                 `json:"source"`
		Target     string                 `json:"target"`
		Healthy    bool                   `json:"healthy,omitempty"`
		CustomData map[string]interface{} `json:"customData,omitempty"`
	} `json:"data"`
}

// NodeType is a Node entity type
type NodeType string

const (
	clusterNT   NodeType = "cluster"
	interfaceNT NodeType = "interface"
	loopConIfNT NodeType = "loopConIfNT"
	clientNT    NodeType = "client"
	forwarderNT NodeType = "forwarder"
	managerNT   NodeType = "manager"
	endpointNT  NodeType = "endpoint"
	serviceNT   NodeType = "service"
	// registryNT  NodeType = "registry"
)

// EdgeType is a connection type
type EdgeType string

const (
	interfaceConnection       EdgeType = "interfaceConnection"
	interfaceCrossConnection  EdgeType = "interfaceCrossConnection"
	interfaceLoopedConnection EdgeType = "interfaceLoopedConnection"
	// serviceRequest           EdgeType = "serviceRequest"
	// registryRequest          EdgeType = "registryRequest"
)

const (
	clientInterface string = "client_interface"
	serverInterface string = "server_interface"
)
