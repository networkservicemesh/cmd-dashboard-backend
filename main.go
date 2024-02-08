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
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
	"github.com/networkservicemesh/sdk/pkg/tools/spiffejwt"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
	"github.com/networkservicemesh/sdk/pkg/tools/tracing"
)

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	// Setup logger
	log.EnableTracing(true)
	logrus.SetFormatter(&nested.Formatter{})
	ctx = log.WithLog(ctx, logruslogger.New(ctx, map[string]interface{}{"cmd": os.Args[:1]}))
	logger := log.FromContext(ctx)

	// Setup Spire
	source, err := workloadapi.NewX509Source(ctx)
	if err != nil {
		logger.Fatalf("Error getting x509 source: %v", err.Error())
	}
	var svid *x509svid.SVID
	svid, err = source.GetX509SVID()
	if err != nil {
		logger.Fatalf("Error getting x509 svid: %v", err.Error())
	}
	logger.Infof("sVID: %q", svid.ID)

	tlsClientConfig := tlsconfig.MTLSClientConfig(source, source, tlsconfig.AuthorizeAny())
	tlsClientConfig.MinVersion = tls.VersionTLS12

	// Configure and run REST API server
	configureAndRunRESTServer()

	// Create gRPC dial options
	dialOptions := append(tracing.WithTracingDial(),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
			grpc.PerRPCCredentials(token.NewPerRPCCredentials(spiffejwt.TokenGeneratorFunc(source, time.Duration(10)*time.Minute))),
		),
		grpc.WithTransportCredentials(credentials.NewTLS(tlsClientConfig)),
	)

	// Monitor Connections
	for ; ctx.Err() == nil; time.Sleep(time.Millisecond * 100) {
		var nseChannel = getNseChannel(ctx, logger, dialOptions)

		for ctx.Err() == nil {
			select {
			case <-ctx.Done():
				return
			case nse, ok := <-nseChannel:
				if !ok {
					break
				}

				nsmgrAddr := strings.Join(strings.Split(strings.TrimPrefix(nse.GetNetworkServiceEndpoint().Url, "tcp://["), "]:"), ":")

				if _, exists := nsmgrs.Load(nsmgrAddr); !exists {
					nsmgrs.Store(nsmgrAddr, true)
					logger.Infof("Extracted NSMGR addr: %q", nsmgrAddr)

					// Start a goroutine for each nsmgr
					go func() {
						nsmgrConn, err := grpc.Dial(nsmgrAddr, dialOptions...)
						if err != nil {
							logger.Errorf("Failed dial to NSMgr %q: %v", nsmgrAddr, err.Error())
							return
						}

						clientConnections := networkservice.NewMonitorConnectionClient(nsmgrConn)

						stream, err := clientConnections.MonitorConnections(ctx, &networkservice.MonitorScopeSelector{})
						if err != nil {
							logger.Errorf("Error from MonitorConnectionClient: %v", err.Error())
							return
						}

						for ctx.Err() == nil {
							event, err := stream.Recv()
							if err != nil {
								logger.Errorf("Error from %q nsmgr monitorConnection stream: %v", nsmgrAddr, err.Error())
								time.Sleep(time.Millisecond * 500)
								cleanupManager(logger, nsmgrAddr)
								break
							}
							updateConnections(logger, nsmgrAddr, event)
						}
					}()
				}
			}
		}
	}
}

func getNseChannel(ctx context.Context, logger log.Logger, dialOptions []grpc.DialOption) <-chan *registry.NetworkServiceEndpointResponse {
	registryAddr := os.Getenv("NSM_REGISTRY_URL")

	if registryAddr == "" {
		registryAddr = "registry:5002"
	}

	conn, err := grpc.Dial(registryAddr, dialOptions...)
	if err != nil {
		logger.Errorf("Failed dial to Registry: %v", err.Error())
	}

	clientNse := registry.NewNetworkServiceEndpointRegistryClient(conn)

	streamNse, err := clientNse.Find(ctx, &registry.NetworkServiceEndpointQuery{
		Watch: true,
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceNames: []string{"forwarder"},
		},
	})
	if err != nil {
		logger.Errorf("Failed to perform Find NSE request: %v", err)
	}

	return registry.ReadNetworkServiceEndpointChannel(streamNse)
}

func updateConnections(logger log.Logger, nsmgrAddr string, event *networkservice.ConnectionEvent) {
	for _, connection := range event.Connections {
		connectionID := generateConnectionID(connection)
		logger.Infof("Handling %q event on %q nsmgr connectionId: %q Connection: %q", event.Type.String(), nsmgrAddr, connectionID, connection)
		switch {
		case event.Type == networkservice.ConnectionEventType_DELETE:
			connections.Delete(connectionID)
			removeManagerConnection(nsmgrAddr, connectionID)
		default: // INITIAL_STATE_TRANSFER and UPDATE event types
			connections.Store(connectionID, connection)
			addManagerConnection(nsmgrAddr, connectionID)
		}
	}
	parceConnectionsToGraphicalModel(logger)
}

func cleanupManager(logger log.Logger, nsmgrAddr string) {
	mgrConnections, _ := managerConnections.Load(nsmgrAddr)
	mgrConnections.Range(func(connectionID string, _ bool) bool {
		logger.Infof("Removing %q nsmgr connection: %q", nsmgrAddr, connectionID)
		connections.Delete(connectionID)
		return true
	})
	managerConnections.Delete(nsmgrAddr)
	nsmgrs.Delete(nsmgrAddr)
	parceConnectionsToGraphicalModel(logger)
}

func configureAndRunRESTServer() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Next()
	})

	r.GET("/nodes", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, storageData.Nodes)
	})
	r.GET("/edges", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, storageData.Edges)
	})

	go func() {
		if err := r.Run(":3001"); err != nil {
			panic("Failed to run REST server: " + err.Error())
		}
	}()
}
