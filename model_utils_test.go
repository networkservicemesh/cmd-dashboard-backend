// Copyright (c) 2024 Pragmagic Inc. and/or its affiliates.
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
	"fmt"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func Test_Interface_Order(t *testing.T) {
	t.Cleanup(func() {
		goleak.VerifyNone(t)
	})

	const nsmgrAddr = "10.244.1.4:5001"
	const nscName = "alpine-0a9d9aa7"
	const cifNscName = "KERNEL/nsm-1"
	const fwd1Name = "forwarder-vpp-t2jgb"
	const cifFwd1Name = "WIREGUARD TUNNEL/wg0"
	const sifFwd1Name = "VIRTIO/tun0"
	const fwd2Name = "forwarder-vpp-9dxwt"
	const cifFwd2Name = "VIRTIO/tun0"
	const sifFwd2Name = "WIREGUARD TUNNEL/wg0"
	const nseName = "nse-kernel-zz6m5"
	const sifNseName = "KERNEL/kernel2ip2-ff6c"

	connection := &networkservice.Connection{
		NetworkService: "kernel2ip2kernel",
		Path: &networkservice.Path{
			PathSegments: []*networkservice.PathSegment{
				{
					Name: nscName,
					Id:   "alpine-0a9d9aa7-0",
					Metrics: map[string]string{
						"client_interface": cifNscName,
					},
				},
				{
					Name: "nsmgr-2pctm",
					Id:   "12345",
				},
				{
					Name: fwd1Name,
					Id:   "23456",
					Metrics: map[string]string{
						"client_interface": cifFwd1Name,
						"server_interface": sifFwd1Name,
					},
				},
				{
					Name: "nsmgr-zjr6b",
					Id:   "34567",
				},
				{
					Name: fwd2Name,
					Id:   "45678",
					Metrics: map[string]string{
						"client_interface": cifFwd2Name,
						"server_interface": sifFwd2Name,
					},
				},
				{
					Name: nseName,
					Id:   "56789",
					Metrics: map[string]string{
						"server_interface": sifNseName,
					},
				},
			},
		},
	}
	connectionID := generateConnectionID(connection)

	connections.Store(connectionID, connection)
	addManagerConnection(nsmgrAddr, connectionID)
	parceConnectionsToGraphicalModel()

	require.Equal(t, 14, len(storageData.Nodes))
	require.Equal(t, 5, len(storageData.Edges))

	// Connection 1 (client-forwarder1)
	source1 := fmt.Sprintf("int-c--%s--%s--%s", connectionID, nscName, cifNscName)
	target1 := fmt.Sprintf("int-s--%s--%s--%s", connectionID, fwd1Name, sifFwd1Name)
	require.Equal(t, source1, storageData.Edges[0].Data.Source)
	require.Equal(t, target1, storageData.Edges[0].Data.Target)
	// Connection 2 (forwarder1 internal interface cross connection)
	source2 := fmt.Sprintf("int-s--%s--%s--%s", connectionID, fwd1Name, sifFwd1Name)
	target2 := fmt.Sprintf("int-c--%s--%s--%s", connectionID, fwd1Name, cifFwd1Name)
	require.Equal(t, source2, storageData.Edges[1].Data.Source)
	require.Equal(t, target2, storageData.Edges[1].Data.Target)
	// Connection 3 (forwarder1-forwarder2)
	source3 := fmt.Sprintf("int-c--%s--%s--%s", connectionID, fwd1Name, cifFwd1Name)
	target3 := fmt.Sprintf("int-s--%s--%s--%s", connectionID, fwd2Name, sifFwd2Name)
	require.Equal(t, source3, storageData.Edges[2].Data.Source)
	require.Equal(t, target3, storageData.Edges[2].Data.Target)
	// Connection 4 (forwarder2 internal interface cross connection)
	source4 := fmt.Sprintf("int-s--%s--%s--%s", connectionID, fwd2Name, sifFwd2Name)
	target4 := fmt.Sprintf("int-c--%s--%s--%s", connectionID, fwd2Name, cifFwd2Name)
	require.Equal(t, source4, storageData.Edges[3].Data.Source)
	require.Equal(t, target4, storageData.Edges[3].Data.Target)
	// Connection 5 (forwarder2-endpoint)
	source5 := fmt.Sprintf("int-c--%s--%s--%s", connectionID, fwd2Name, cifFwd2Name)
	target5 := fmt.Sprintf("int-s--%s--%s--%s", connectionID, nseName, sifNseName)
	require.Equal(t, source5, storageData.Edges[4].Data.Source)
	require.Equal(t, target5, storageData.Edges[4].Data.Target)

	// Cleanup
	connections.Delete(connectionID)
	managerConnections.Delete(nsmgrAddr)
	nsmgrs.Delete(nsmgrAddr)
}

func Test_NSE_To_NSE_Connection(t *testing.T) {
	t.Cleanup(func() {
		goleak.VerifyNone(t)
	})

	const nsmgrAddr = "10.244.1.5:5001"

	connection := &networkservice.Connection{
		NetworkService: "ns1",
		Path: &networkservice.Path{
			PathSegments: []*networkservice.PathSegment{
				{
					Name: "nse-vl3-vpp-9ln7f",
					Id:   "12345",
					Metrics: map[string]string{
						"server_interface": "MEMIF/memif2670642822/2",
					},
				},
				{
					Name: "nsmgr-j6x6m",
					Id:   "23456",
				},
				{
					Name: "forwarder-vpp-zpg6b",
					Id:   "34567",
					Metrics: map[string]string{
						"client_interface": "WIREGUARD TUNNEL/wg1",
						"server_interface": "MEMIF/memif2670642822/0",
					},
				},
				{
					Name: "nsmgr-rp4rr",
					Id:   "45678",
				},
				{
					Name: "forwarder-vpp-s7sw5",
					Id:   "56789",
					Metrics: map[string]string{
						"client_interface": "MEMIF/memif2670642822/0",
						"server_interface": "WIREGUARD TUNNEL/wg1",
					},
				},
				{
					Name: "nse-vl3-vpp-gcf6j",
					Id:   "67890",
					Metrics: map[string]string{
						"server_interface": "MEMIF/memif2670642822/1",
					},
				},
			},
		},
	}
	connectionID := generateConnectionID(connection)

	connections.Store(connectionID, connection)
	addManagerConnection(nsmgrAddr, connectionID)
	parceConnectionsToGraphicalModel()

	// 14: 1 cluster + 1 ns + 2 nse + 2 mgr + 2 fwd + 6 if
	require.Equal(t, 14, len(storageData.Nodes))
	require.Equal(t, 5, len(storageData.Edges))

	// Check that all connections have sources and targets
	for _, edge := range storageData.Edges {
		require.NotEmpty(t, edge.Data.Source)
		require.NotEmpty(t, edge.Data.Target)
	}

	// Cleanup
	connections.Delete(connectionID)
	managerConnections.Delete(nsmgrAddr)
	nsmgrs.Delete(nsmgrAddr)
}

func Test_Looped_NSE_To_NSE_Connection(t *testing.T) {
	t.Cleanup(func() {
		goleak.VerifyNone(t)
	})

	const nsmgrAddr = "10.244.1.6:5001"

	connection := &networkservice.Connection{
		NetworkService: "ns2",
		Path: &networkservice.Path{
			PathSegments: []*networkservice.PathSegment{
				{
					Name: "nse-vl3-vpp-gcf6j",
					Id:   "12345",
					Metrics: map[string]string{
						"server_interface": "MEMIF/memif2670642822/2",
					},
				},
				{
					Name: "nsmgr-rp4rr",
					Id:   "23456",
				},
				{
					Name: "forwarder-vpp-s7sw5",
					Id:   "34567",
					Metrics: map[string]string{
						"client_interface": "MEMIF/memif3519870697/0",
						"server_interface": "VIRTIO/tun0",
					},
				},
				{
					Name: "nse-vl3-vpp-gcf6j",
					Id:   "12345",
					Metrics: map[string]string{
						"server_interface": "MEMIF/memif2670642822/1",
					},
				},
			},
		},
	}
	connectionID := generateConnectionID(connection)

	connections.Store(connectionID, connection)
	addManagerConnection(nsmgrAddr, connectionID)
	parceConnectionsToGraphicalModel()

	// 9: 1 cluster + 1 ns + 1 nse + 1 mgr + 1 fwd + 4 if
	require.Equal(t, 9, len(storageData.Nodes))
	require.Equal(t, 3, len(storageData.Edges))

	// Check that all connections have sources and targets and a proper type
	for _, edge := range storageData.Edges {
		require.NotEmpty(t, edge.Data.Source)
		require.NotEmpty(t, edge.Data.Target)
		require.Equal(t, interfaceLoopedConnection, edge.Data.Type)
	}

	// Cleanup
	connections.Delete(connectionID)
	managerConnections.Delete(nsmgrAddr)
	nsmgrs.Delete(nsmgrAddr)
}
