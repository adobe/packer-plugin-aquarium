/**
 * Copyright 2025 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 */

// Author: Sergei Parshev (@sparshev)

package aquarium

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	connect "connectrpc.com/connect"
	aquariumv2 "github.com/adobe/aquarium-fish/lib/rpc/proto/aquarium/v2"
	aquariumv2connect "github.com/adobe/aquarium-fish/lib/rpc/proto/aquarium/v2/aquariumv2connect"
)

// API Client for AquariumFish
type APIClient struct {
	BaseURL string

	// underlying HTTP client used by connect clients (injects Basic Auth)
	httpClient connectHTTPClient

	// generated RPC clients
	labelClient     aquariumv2connect.LabelServiceClient
	appClient       aquariumv2connect.ApplicationServiceClient
	userClient      aquariumv2connect.UserServiceClient
	gateProxySSH    aquariumv2connect.GateProxySSHServiceClient
	streamingClient aquariumv2connect.StreamingServiceClient
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL, username, password string, httpClient *http.Client) *APIClient {
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Prepare a connect-compatible HTTP client that injects Basic auth
	auth := basicAuth(username, password)
	ch := connectHTTPClient{base: httpClient, authHeader: auth}

	c := &APIClient{BaseURL: baseURL, httpClient: ch}
	c.labelClient = aquariumv2connect.NewLabelServiceClient(ch, baseURL)
	c.appClient = aquariumv2connect.NewApplicationServiceClient(ch, baseURL)
	c.userClient = aquariumv2connect.NewUserServiceClient(ch, baseURL)
	c.gateProxySSH = aquariumv2connect.NewGateProxySSHServiceClient(ch, baseURL)
	c.streamingClient = aquariumv2connect.NewStreamingServiceClient(ch, baseURL)
	return c
}

// connectHTTPClient injects Authorization header for all requests
type connectHTTPClient struct {
	base       *http.Client
	authHeader string
}

func (c connectHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if c.authHeader != "" {
		req.Header.Set("Authorization", c.authHeader)
	}
	return c.base.Do(req)
}

func basicAuth(user, pass string) string {
	token := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	return "Basic " + token
}

// GetLabels retrieves labels, optionally filtered by name and version
func (c *APIClient) GetLabels(ctx context.Context, name, version string) ([]*aquariumv2.Label, error) {
	req := aquariumv2.LabelServiceListRequest{}
	if name != "" {
		req.Name = &name
	}
	if version != "" {
		req.Version = &version
	}
	resp, err := c.labelClient.List(ctx, connectRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetData(), nil
}

// CreateApplication creates a new application
func (c *APIClient) CreateApplication(ctx context.Context, app *aquariumv2.Application) (*aquariumv2.Application, error) {
	resp, err := c.appClient.Create(ctx, connectRequest(aquariumv2.ApplicationServiceCreateRequest{Application: app}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetData(), nil
}

// GetApplicationState retrieves the current state of an application
func (c *APIClient) GetApplicationState(ctx context.Context, uid string) (*aquariumv2.ApplicationState, error) {
	resp, err := c.appClient.GetState(ctx, connectRequest(aquariumv2.ApplicationServiceGetStateRequest{ApplicationUid: uid}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetData(), nil
}

// GetApplicationResource retrieves the application resource
func (c *APIClient) GetApplicationResource(ctx context.Context, uid string) (*aquariumv2.ApplicationResource, error) {
	resp, err := c.appClient.GetResource(ctx, connectRequest(aquariumv2.ApplicationServiceGetResourceRequest{ApplicationUid: uid}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetData(), nil
}

// GetApplicationResourceAccess retrieves SSH access credentials
func (c *APIClient) GetApplicationResourceAccess(ctx context.Context, resourceUID string) (*aquariumv2.GateProxySSHAccess, error) {
	// Receiving static credential because Packer has no proper mechanism to use OTP
	static := true
	resp, err := c.gateProxySSH.GetResourceAccess(ctx, connectRequest(aquariumv2.GateProxySSHServiceGetResourceAccessRequest{
		ApplicationResourceUid: resourceUID,
		Static:                 &static,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetData(), nil
}

// DeallocateApplication triggers application deallocation
func (c *APIClient) DeallocateApplication(ctx context.Context, uid string) error {
	_, err := c.appClient.Deallocate(ctx, connectRequest(aquariumv2.ApplicationServiceDeallocateRequest{ApplicationUid: uid}))
	return err
}

// CreateApplicationTask creates a new application task
func (c *APIClient) CreateApplicationTask(ctx context.Context, task *aquariumv2.ApplicationTask) (*aquariumv2.ApplicationTask, error) {
	resp, err := c.appClient.CreateTask(ctx, connectRequest(aquariumv2.ApplicationServiceCreateTaskRequest{Task: task}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetData(), nil
}

// GetApplicationTask retrieves an application task
func (c *APIClient) GetApplicationTask(ctx context.Context, taskUID string) (*aquariumv2.ApplicationTask, error) {
	resp, err := c.appClient.GetTask(ctx, connectRequest(aquariumv2.ApplicationServiceGetTaskRequest{ApplicationTaskUid: taskUID}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetData(), nil
}

// Subscribe opens a server stream for database change notifications
func (c *APIClient) Subscribe(ctx context.Context, types []aquariumv2.SubscriptionType) (*streamWrapper, error) {
	req := aquariumv2.StreamingServiceSubscribeRequest{SubscriptionTypes: types}
	stream, err := c.streamingClient.Subscribe(ctx, connectRequest(req))
	if err != nil {
		return nil, err
	}
	return &streamWrapper{stream: stream}, nil
}

type streamWrapper struct {
	stream *connect.ServerStreamForClient[aquariumv2.StreamingServiceSubscribeResponse]
}

func (s *streamWrapper) Receive() (*aquariumv2.StreamingServiceSubscribeResponse, error) {
	if !s.stream.Receive() {
		return nil, s.stream.Err()
	}
	return s.stream.Msg(), nil
}

func (s *streamWrapper) Close() error { return s.stream.Close() }

// connectRequest is a small helper to avoid importing connect in every caller
// Note: some generated clients expect *connect.Request[T] where T is a concrete message type
// Using pointer where the client expects non-pointer causes a mismatch. Provide overloads as needed.
func connectRequest[T any](msg T) *connect.Request[T] { return connect.NewRequest[T](&msg) }

// ParseSSHAddress parses SSH address into host and port
func ParseSSHAddress(addr string) (string, int, error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid SSH address format: %s", addr)
	}

	host := parts[0]
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid port in SSH address: %s", parts[1])
	}

	return host, port, nil
}

// GetCurrentUser retrieves the current authenticated user (used as connectivity check)
func (c *APIClient) GetCurrentUser(ctx context.Context) (*aquariumv2.User, error) {
	resp, err := c.userClient.GetMe(ctx, connectRequest(aquariumv2.UserServiceGetMeRequest{}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetData(), nil
}
