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

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/edwarnicke/genericsync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

func updateStorage(nodes []Node, edges []Edge) {
	storageData.Nodes = nodes
	storageData.Edges = edges
}

func addManagerConnection(mgr, conn string) {
	if _, ok := managerConnections.Load(mgr); !ok {
		managerConnections.Store(mgr, new(genericsync.Map[string, bool]))
	}
	conns, _ := managerConnections.Load(mgr)
	conns.Store(conn, true)
	managerConnections.Store(mgr, conns)
}

func removeManagerConnection(mgr, conn string) {
	if conns, ok := managerConnections.Load(mgr); ok {
		conns.Delete(conn)
		if isMapEmpty(conns) {
			managerConnections.Delete(mgr)
		}
	}
}

func parceConnectionsToGraphicalModel(logger log.Logger) {
	nodeMap := make(map[string]Node)
	var edges []Edge

	// Add local cluster node
	var clusterLocal Node
	if !isConnectionMapEmpty(&connections) {
		clusterLocal = makeLocalCluster(nodeMap)
	}

	connections.Range(func(connectionID string, conn *networkservice.Connection) bool {
		healthy := conn.GetState() != networkservice.State_DOWN

		// Add Network Service Node
		ns := makeNetworkService(nodeMap, conn.GetNetworkService(), clusterLocal.Data.ID)

		// Create all path segment nodes, interfaces and interface connections
		pathSegments := conn.GetPath().GetPathSegments()

		// Skip looped endpoint-endpoint connections (vl3 scenario)
		if pathSegments[0].GetName() == pathSegments[len(pathSegments)-1].GetName() {
			logger.Infof("Connection %q looped on %q is skipped from conversion to graphical model.", connectionID, pathSegments[0].GetName())
			return true
		}

		var previousInterfaceID string
		for _, segment := range pathSegments {
			segmentType := getPathSegmentType(segment.GetName())

			// Add segment node
			node := makeSegmentNode(nodeMap, segment.GetName(), clusterLocal.Data.ID, ns.Data.ID, segmentType)

			switch {
			case segmentType == clientNT:
				interfID := fmt.Sprintf("int-c--%s--%s", connectionID, node.Data.ID)
				nodeMap[interfID] = makeInterface(interfID, node.Data.ID, getInterfaceLabelFromMetrics(segment, clientInterface))
				previousInterfaceID = interfID
			case segmentType == endpointNT:
				interfID := fmt.Sprintf("int-s--%s--%s", connectionID, node.Data.ID)
				nodeMap[interfID] = makeInterface(interfID, node.Data.ID, getInterfaceLabelFromMetrics(segment, serverInterface))
				if previousInterfaceID != "" {
					edges = addEdge(edges, previousInterfaceID, interfID, interfaceConnection, healthy)
				}
				previousInterfaceID = interfID
			case segmentType == forwarderNT:
				interfCID := fmt.Sprintf("int-c--%s--%s", connectionID, node.Data.ID)
				nodeMap[interfCID] = makeInterface(interfCID, node.Data.ID, getInterfaceLabelFromMetrics(segment, clientInterface))
				edges = addEdge(edges, previousInterfaceID, interfCID, interfaceConnection, healthy)
				interfEID := fmt.Sprintf("int-s--%s--%s", connectionID, node.Data.ID)
				nodeMap[interfEID] = makeInterface(interfEID, node.Data.ID, getInterfaceLabelFromMetrics(segment, serverInterface))
				edges = addEdge(edges, interfCID, interfEID, interfaceCrossConnection, healthy)
				previousInterfaceID = interfEID
				// TODO Aggregate statistics for the Overview page
			}
		}
		return true
	})

	edges = addInternalNSEInterfaceConnections(edges, nodeMap)

	updateStorage(mapToArray(nodeMap), edges)
}

func addInternalNSEInterfaceConnections(edges []Edge, nodeMap map[string]Node) []Edge {
	var endpoints []Node
	for _, node := range nodeMap {
		if node.Data.Type == endpointNT {
			endpoints = append(endpoints, node)
		}
	}
	for _, nse := range endpoints {
		var interfaces []Node
		for _, node := range nodeMap {
			if node.Data.Parent == nse.Data.ID {
				interfaces = append(interfaces, node)
			}
		}
		if len(interfaces) > 1 {
			for i := 0; i < len(interfaces)-1; i++ {
				for j := i + 1; j < len(interfaces)-1; j++ {
					edges = addEdge(edges, interfaces[i].Data.ID, interfaces[j].Data.ID, interfaceCrossConnection, true)
				}
			}
		}
	}
	return edges
}

func makeLocalCluster(nodeMap map[string]Node) Node {
	clusterLocal := Node{}
	clusterLocalName := "cluster-local"
	clusterLocal.Data.ID = clusterLocalName
	clusterLocal.Data.Type = clusterNT
	clusterLocal.Data.Label = clusterLocalName
	nodeMap[clusterLocal.Data.ID] = clusterLocal
	return clusterLocal
}

func makeNetworkService(nodeMap map[string]Node, id, parentID string) Node {
	ns := Node{}
	ns.Data.ID = id
	if !duplicateNodeExists(nodeMap, ns.Data.ID) {
		ns.Data.Type = serviceNT
		ns.Data.Label = id
		ns.Data.Parent = parentID
		nodeMap[ns.Data.ID] = ns
	}
	return ns
}

func makeSegmentNode(nodeMap map[string]Node, id, parentID, nsID string, segmentType NodeType) Node {
	node := Node{}
	node.Data.ID = id
	if !duplicateNodeExists(nodeMap, node.Data.ID) {
		node.Data.Type = segmentType
		node.Data.Parent = parentID
		if node.Data.Type == endpointNT {
			node.Data.CustomData = make(map[string]interface{})
			node.Data.CustomData["networkService"] = nsID
		}
		node.Data.Label = id
		nodeMap[node.Data.ID] = node
	}
	return node
}

func makeInterface(id, parentID, label string) Node {
	interf := Node{}
	interf.Data.ID = id
	interf.Data.Type = interfaceNT
	interf.Data.Parent = parentID
	interf.Data.Label = label
	return interf
}

func addEdge(edges []Edge, sourceID, targetID string, edgeType EdgeType, healthy bool) []Edge {
	edge := Edge{}
	edge.Data.ID = fmt.Sprintf("conn--%s--%s", sourceID, targetID)
	edge.Data.Type = edgeType
	edge.Data.Source = sourceID
	edge.Data.Target = targetID
	edge.Data.Healthy = healthy
	return append(edges, edge)
}

func getInterfaceLabelFromMetrics(segment *networkservice.PathSegment, interfaceType string) string {
	metrics := segment.GetMetrics()
	if metrics != nil && metrics[interfaceType] != "" {
		return metrics[interfaceType]
	}
	return "unknown"
}

func duplicateNodeExists(nodeMap map[string]Node, id string) bool {
	if _, exists := nodeMap[id]; exists {
		return true
	}
	return false
}

func mapToArray(nodeMap map[string]Node) []Node {
	nodes := make([]Node, 0, len(nodeMap))
	for _, node := range nodeMap {
		nodes = append(nodes, node)
	}
	return nodes
}

func getPathSegmentType(name string) NodeType {
	switch {
	case strings.HasPrefix(name, "nsmgr-"):
		return managerNT
	case strings.HasPrefix(name, "forwarder-") || strings.HasPrefix(name, "fwd-"):
		return forwarderNT
	case strings.HasPrefix(name, "nse-"):
		return endpointNT
	default:
		return clientNT
	}
}

func generateConnectionID(connection *networkservice.Connection) string {
	ids := ""
	for _, segment := range connection.GetPath().GetPathSegments() {
		ids += segment.GetId()
	}
	return generateHash(ids)
}

func generateHash(input string) string {
	hasher := sha256.New()
	hasher.Write([]byte(input))
	hashBytes := hasher.Sum(nil)
	hash := hex.EncodeToString(hashBytes)
	return hash
}

func isMapEmpty[K string, V any](m *genericsync.Map[K, V]) bool {
	isEmpty := true
	m.Range(func(k K, v V) bool {
		isEmpty = false
		return false
	})
	return isEmpty
}

func isConnectionMapEmpty[K string, V *networkservice.Connection](m *genericsync.Map[K, V]) bool {
	isEmpty := true
	m.Range(func(k K, v V) bool {
		isEmpty = false
		return false
	})
	return isEmpty
}
