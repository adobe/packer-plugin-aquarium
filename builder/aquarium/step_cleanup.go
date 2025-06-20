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
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepCleanup handles cleanup of AquariumFish resources
type StepCleanup struct {
	Config     *Config
	HTTPClient *http.Client
}

// Run executes the cleanup step
func (s *StepCleanup) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	return multistep.ActionContinue
}

// Cleanup deallocates the application if it was allocated
func (s *StepCleanup) Cleanup(state multistep.StateBag) {
	ui := state.Get("ui").(packersdk.Ui)

	// Get the API client if available
	client, hasClient := state.GetOk("api_client")
	if !hasClient {
		ui.Say("No API client found, skipping cleanup")
		return
	}
	apiClient := client.(*APIClient)

	// Get the application if available
	app, hasApp := state.GetOk("application")
	if !hasApp {
		ui.Say("No application found, skipping cleanup")
		return
	}
	application := app.(*Application)

	ui.Say("Cleaning up AquariumFish resources...")

	// Trigger application deallocation
	err := apiClient.DeallocateApplication(application.UID)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to deallocate application: %v", err))
		// Don't halt on cleanup errors, just log them
		return
	}

	ui.Say(fmt.Sprintf("Application %s deallocate request sent...", application.UID))

	// Wait a bit to ensure deallocation starts
	time.Sleep(5 * time.Second)

	// Optionally wait for deallocation to complete
	ui.Say("Waiting for deallocation to complete...")
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			ui.Say("Deallocation timeout reached, but continuing...")
			return

		case <-ticker.C:
			// Check application state
			appState, err := apiClient.GetApplicationState(application.UID)
			if err != nil {
				ui.Say(fmt.Sprintf("Could not check application state: %v", err))
				return
			}

			ui.Say(fmt.Sprintf("Application status: %s", appState.Status))

			if appState.Status == "DEALLOCATED" || appState.Status == "RECALLED" {
				ui.Say("Application successfully deallocated")
				return
			}

			if appState.Status == "ERROR" {
				ui.Say(fmt.Sprintf("Application in error state during deallocation: %s", appState.Description))
				return
			}

			// Continue waiting for other statuses
		}
	}
}
