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

	nsmgrAddr := "10.244.1.4:5001"
	nscName := "alpine-0a9d9aa7"
	fwd1Name := "forwarder-vpp-t2jgb"
	fwd2Name := "forwarder-vpp-9dxwt"
	nseName := "nse-kernel-zz6m5"

	connection := &networkservice.Connection{
		NetworkService: "kernel2ip2kernel",
		Path: &networkservice.Path{
			PathSegments: []*networkservice.PathSegment{
				{
					Name: nscName,
					Id:   "alpine-0a9d9aa7-0",
					Metrics: map[string]string{
						"client_interface": "KERNEL/nsm-1",
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
						"client_interface": "WIREGUARD TUNNEL/wg0",
						"server_interface": "VIRTIO/tun0",
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
						"client_interface": "VIRTIO/tun0",
						"server_interface": "WIREGUARD TUNNEL/wg0",
					},
				},
				{
					Name: nseName,
					Id:   "56789",
					Metrics: map[string]string{
						"server_interface": "KERNEL/kernel2ip2-ff6c",
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
	source1 := fmt.Sprintf("int-c--%s--%s", connectionID, nscName)
	target1 := fmt.Sprintf("int-s--%s--%s", connectionID, fwd1Name)
	require.Equal(t, source1, storageData.Edges[0].Data.Source)
	require.Equal(t, target1, storageData.Edges[0].Data.Target)
	// Connection 2 (forwarder1 internal interface cross connection)
	source2 := fmt.Sprintf("int-s--%s--%s", connectionID, fwd1Name)
	target2 := fmt.Sprintf("int-c--%s--%s", connectionID, fwd1Name)
	require.Equal(t, source2, storageData.Edges[1].Data.Source)
	require.Equal(t, target2, storageData.Edges[1].Data.Target)
	// Connection 3 (forwarder1-forwarder2)
	source3 := fmt.Sprintf("int-c--%s--%s", connectionID, fwd1Name)
	target3 := fmt.Sprintf("int-s--%s--%s", connectionID, fwd2Name)
	require.Equal(t, source3, storageData.Edges[2].Data.Source)
	require.Equal(t, target3, storageData.Edges[2].Data.Target)
	// Connection 4 (forwarder2 internal interface cross connection)
	source4 := fmt.Sprintf("int-s--%s--%s", connectionID, fwd2Name)
	target4 := fmt.Sprintf("int-c--%s--%s", connectionID, fwd2Name)
	require.Equal(t, source4, storageData.Edges[3].Data.Source)
	require.Equal(t, target4, storageData.Edges[3].Data.Target)
	// Connection 5 (forwarder2-endpoint)
	source5 := fmt.Sprintf("int-c--%s--%s", connectionID, fwd2Name)
	target5 := fmt.Sprintf("int-s--%s--%s", connectionID, nseName)
	require.Equal(t, source5, storageData.Edges[4].Data.Source)
	require.Equal(t, target5, storageData.Edges[4].Data.Target)

	// Cleanup
	connections.Delete(connectionID)
	managerConnections.Delete(nsmgrAddr)
	nsmgrs.Delete(nsmgrAddr)
	parceConnectionsToGraphicalModel()

	require.Equal(t, 0, len(storageData.Nodes))
	require.Equal(t, 0, len(storageData.Edges))
}
