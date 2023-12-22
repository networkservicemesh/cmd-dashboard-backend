// Copyright (c) 2023 Cisco and/or its affiliates.
//
// Copyright (c) 2023 Pragmagic Inc. and/or its affiliates.
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
	"github.com/edwarnicke/genericsync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

// StorageData is a current graphical model state
type StorageData struct {
	Nodes []Node
	Edges []Edge
}

var storageData StorageData
var connections genericsync.Map[string, *networkservice.Connection]
var managerConnections genericsync.Map[string, *genericsync.Map[string, bool]]
var nsmgrs genericsync.Map[string, bool]
