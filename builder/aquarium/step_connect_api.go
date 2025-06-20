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
	"fmt"
	"net/http"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepConnectAPI connects to the AquariumFish API and verifies authentication
type StepConnectAPI struct {
	Config     *Config
	HTTPClient *http.Client
}

// Run executes the step to connect to the API
func (s *StepConnectAPI) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Connecting to AquariumFish API...")

	// Create API client
	client := NewAPIClient(s.Config.Endpoint, s.Config.Username, s.Config.Password, s.HTTPClient)

	// Test the connection by getting the current user info
	resp, err := client.makeRequest("GET", "/api/v1/user/me/", nil)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to connect to AquariumFish API: %v", err))
		state.Put("error", fmt.Errorf("API connection failed: %v", err))
		return multistep.ActionHalt
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		ui.Error("Authentication failed. Please check your credentials.")
		state.Put("error", fmt.Errorf("authentication failed"))
		return multistep.ActionHalt
	}

	if resp.StatusCode != http.StatusOK {
		ui.Error(fmt.Sprintf("API connection test failed with status %d", resp.StatusCode))
		state.Put("error", fmt.Errorf("API connection test failed"))
		return multistep.ActionHalt
	}

	ui.Say("Successfully connected to AquariumFish API")

	// Store the API client in state for other steps
	state.Put("api_client", client)

	return multistep.ActionContinue
}

// Cleanup performs any necessary cleanup
func (s *StepConnectAPI) Cleanup(state multistep.StateBag) {
	// Nothing to clean up for API connection
}
