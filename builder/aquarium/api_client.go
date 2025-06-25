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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

// API Client for AquariumFish
type APIClient struct {
	BaseURL    string
	Username   string
	Password   string
	HTTPClient *http.Client
}

// Data structures based on OpenAPI specification

type Label struct {
	UID         string            `json:"UID"`
	CreatedAt   time.Time         `json:"created_at"`
	Name        string            `json:"name"`
	Version     int               `json:"version"`
	Definitions []LabelDefinition `json:"definitions"`
	Metadata    map[string]any    `json:"metadata"`
}

type LabelDefinition struct {
	Driver         string          `json:"driver"`
	Resources      Resources       `json:"resources"`
	Options        map[string]any  `json:"options"`
	Authentication *Authentication `json:"authentication,omitempty"`
}

type Resources struct {
	Slots        int                      `json:"slots,omitempty"`
	CPU          int                      `json:"cpu"`
	RAM          int                      `json:"ram"`
	Disks        map[string]ResourcesDisk `json:"disks"`
	Network      string                   `json:"network"`
	NodeFilter   []string                 `json:"node_filter,omitempty"`
	Multitenancy bool                     `json:"multitenancy"`
	CPUOverbook  bool                     `json:"cpu_overbook"`
	RAMOverbook  bool                     `json:"ram_overbook"`
	Lifetime     string                   `json:"lifetime,omitempty"`
}

type ResourcesDisk struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Size  int    `json:"size,omitempty"`
	Reuse bool   `json:"reuse"`
	Clone string `json:"clone,omitempty"`
}

type Authentication struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Key      string `json:"key"`
	Port     int    `json:"port"`
}

type Application struct {
	UID       string         `json:"UID,omitempty"`
	CreatedAt time.Time      `json:"created_at,omitempty"`
	OwnerName string         `json:"owner_name,omitempty"`
	LabelUID  string         `json:"label_UID"`
	Metadata  map[string]any `json:"metadata"`
}

type ApplicationState struct {
	UID            string    `json:"UID"`
	CreatedAt      time.Time `json:"created_at"`
	ApplicationUID string    `json:"application_UID"`
	Status         string    `json:"status"`
	Description    string    `json:"description"`
}

type ApplicationResource struct {
	UID             string          `json:"UID"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	ApplicationUID  string          `json:"application_UID"`
	NodeUID         string          `json:"node_UID"`
	LabelUID        string          `json:"label_UID"`
	DefinitionIndex int             `json:"definition_index"`
	Identifier      string          `json:"identifier"`
	IPAddr          string          `json:"ip_addr"`
	HWAddr          string          `json:"hw_addr"`
	Metadata        map[string]any  `json:"metadata"`
	Authentication  *Authentication `json:"authentication,omitempty"`
	Timeout         time.Time       `json:"timeout"`
}

type ApplicationResourceAccess struct {
	UID                    string    `json:"UID"`
	CreatedAt              time.Time `json:"created_at"`
	ApplicationResourceUID string    `json:"application_resource_UID"`
	Address                string    `json:"address"`
	Username               string    `json:"username"`
	Password               string    `json:"password"`
	Key                    string    `json:"key"`
}

type ApplicationTask struct {
	UID            string         `json:"UID,omitempty"`
	CreatedAt      time.Time      `json:"created_at,omitempty"`
	UpdatedAt      time.Time      `json:"updated_at,omitempty"`
	ApplicationUID string         `json:"application_UID"`
	Task           string         `json:"task"`
	When           string         `json:"when"`
	Options        map[string]any `json:"options,omitempty"`
	Result         map[string]any `json:"result,omitempty"`
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL, username, password string, httpClient *http.Client) *APIClient {
	return &APIClient{
		BaseURL:    strings.TrimSuffix(baseURL, "/"),
		Username:   username,
		Password:   password,
		HTTPClient: httpClient,
	}
}

// makeRequest performs an HTTP request
func (c *APIClient) makeRequest(method, endpoint string, body any) (*http.Response, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %v", err)
	}

	u.Path = path.Join(u.Path, endpoint)
	if strings.HasSuffix(endpoint, "/") {
		u.Path += "/"
	}

	var reqBody io.Reader

	// For non-POST methods, check if body is url.Values and use as query parameters
	if body != nil && method != "POST" {
		if queryParams, ok := body.(*url.Values); ok {
			u.RawQuery = queryParams.Encode()
			body = nil // Clear body since we used it for query params
		}
	}

	// Handle JSON body for POST requests or when body is not url.Values
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, u.String(), reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return c.HTTPClient.Do(req)
}

// GetLabels retrieves labels, optionally filtered by name and version
func (c *APIClient) GetLabels(name, version string) ([]Label, error) {
	endpoint := "/api/v1/label/"

	// Prepare query parameters
	var queryParams *url.Values
	if name != "" || version != "" {
		queryParams = &url.Values{}
		if name != "" {
			queryParams.Add("name", name)
		}
		if version != "" {
			queryParams.Add("version", version)
		}
	}

	resp, err := c.makeRequest("GET", endpoint, queryParams)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var labels []Label
	if err := json.NewDecoder(resp.Body).Decode(&labels); err != nil {
		return nil, fmt.Errorf("failed to decode labels response: %v", err)
	}

	return labels, nil
}

// CreateApplication creates a new application
func (c *APIClient) CreateApplication(app Application) (*Application, error) {
	resp, err := c.makeRequest("POST", "/api/v1/application/", app)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var createdApp Application
	if err := json.NewDecoder(resp.Body).Decode(&createdApp); err != nil {
		return nil, fmt.Errorf("failed to decode application response: %v", err)
	}

	return &createdApp, nil
}

// GetApplicationState retrieves the current state of an application
func (c *APIClient) GetApplicationState(uid string) (*ApplicationState, error) {
	endpoint := fmt.Sprintf("/api/v1/application/%s/state", uid)
	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var state ApplicationState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to decode application state response: %v", err)
	}

	return &state, nil
}

// GetApplicationResource retrieves the application resource
func (c *APIClient) GetApplicationResource(uid string) (*ApplicationResource, error) {
	endpoint := fmt.Sprintf("/api/v1/application/%s/resource", uid)
	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, nil // Resource not found yet
		}
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var resource ApplicationResource
	if err := json.NewDecoder(resp.Body).Decode(&resource); err != nil {
		return nil, fmt.Errorf("failed to decode application resource response: %v", err)
	}

	return &resource, nil
}

// GetApplicationResourceAccess retrieves SSH access credentials
func (c *APIClient) GetApplicationResourceAccess(resourceUID string) (*ApplicationResourceAccess, error) {
	endpoint := fmt.Sprintf("/api/v1/applicationresource/%s/access", resourceUID)
	// Requesting multi-use access to simplify communication logic
	queryParams := &url.Values{}
	queryParams.Add("one_time", "false")
	resp, err := c.makeRequest("GET", endpoint, queryParams)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, nil // Access not available yet
		}
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var access ApplicationResourceAccess
	if err := json.NewDecoder(resp.Body).Decode(&access); err != nil {
		return nil, fmt.Errorf("failed to decode access response: %v", err)
	}

	return &access, nil
}

// DeallocateApplication triggers application deallocation
func (c *APIClient) DeallocateApplication(uid string) error {
	endpoint := fmt.Sprintf("/api/v1/application/%s/deallocate", uid)
	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// CreateApplicationTask creates a new application task
func (c *APIClient) CreateApplicationTask(appUID string, task ApplicationTask) (*ApplicationTask, error) {
	endpoint := fmt.Sprintf("/api/v1/application/%s/task/", appUID)
	resp, err := c.makeRequest("POST", endpoint, task)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var createdTask ApplicationTask
	if err := json.NewDecoder(resp.Body).Decode(&createdTask); err != nil {
		return nil, fmt.Errorf("failed to decode task response: %v", err)
	}

	return &createdTask, nil
}

// GetApplicationTask retrieves an application task
func (c *APIClient) GetApplicationTask(taskUID string) (*ApplicationTask, error) {
	endpoint := fmt.Sprintf("/api/v1/task/%s", taskUID)
	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var task ApplicationTask
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode task response: %v", err)
	}

	return &task, nil
}

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
