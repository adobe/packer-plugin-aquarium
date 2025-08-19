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
	"net/url"
	"time"

	aquariumv2 "github.com/adobe/aquarium-fish/lib/rpc/proto/aquarium/v2"
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
	endpointURL, _ := url.Parse(s.Config.Endpoint)
	if endpointURL.Path == "" {
		// Setting "grpc" if the path is empty
		endpointURL.Path = "grpc"
	}
	client := NewAPIClient(endpointURL.String(), s.Config.Username, s.Config.Password, s.HTTPClient)

	// Test the connection by getting the current user info
	ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if _, err := client.GetCurrentUser(ctxTimeout); err != nil {
		ui.Error(fmt.Sprintf("Failed to connect to AquariumFish API: %v", err))
		state.Put("error", fmt.Errorf("API connection failed: %v", err))
		return multistep.ActionHalt
	}

	ui.Say("Successfully connected to AquariumFish API")

	// Store the API client in state for other steps
	state.Put("api_client", client)

	// Create subscription stream for updates used by later steps
	// Subscribe to objects we care about during build
	subTypes := []aquariumv2.SubscriptionType{
		aquariumv2.SubscriptionType_SUBSCRIPTION_TYPE_APPLICATION,
		aquariumv2.SubscriptionType_SUBSCRIPTION_TYPE_APPLICATION_STATE,
		aquariumv2.SubscriptionType_SUBSCRIPTION_TYPE_APPLICATION_RESOURCE,
		aquariumv2.SubscriptionType_SUBSCRIPTION_TYPE_APPLICATION_TASK,
	}
	stream, err := client.Subscribe(ctx, subTypes)
	if err == nil {
		state.Put("subscribe_stream", stream)
	}

	return multistep.ActionContinue
}

// Cleanup performs any necessary cleanup
func (s *StepConnectAPI) Cleanup(state multistep.StateBag) {
	// Nothing to clean up for API connection
}
