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

// StepWaitForAllocation waits for the application to be allocated
type StepWaitForAllocation struct {
	Config     *Config
	HTTPClient *http.Client
}

// Run executes the step to wait for allocation
func (s *StepWaitForAllocation) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	client := state.Get("api_client").(*APIClient)
	application := state.Get("application").(*Application)

	ui.Say("Waiting for application to be allocated...")

	// Set up timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, s.Config.allocationTimeoutDuration)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastStatus string
	for {
		select {
		case <-timeoutCtx.Done():
			ui.Error(fmt.Sprintf("Allocation timeout reached (%s)", s.Config.AllocationTimeout))
			state.Put("error", fmt.Errorf("allocation timeout"))
			return multistep.ActionHalt

		case <-ticker.C:
			// Get current application state
			appState, err := client.GetApplicationState(application.UID)
			if err != nil {
				ui.Error(fmt.Sprintf("Failed to get application state: %v", err))
				state.Put("error", fmt.Errorf("failed to get application state: %v", err))
				return multistep.ActionHalt
			}

			// Log status changes
			if appState.Status != lastStatus {
				ui.Say(fmt.Sprintf("Application status: %s - %s", appState.Status, appState.Description))
				lastStatus = appState.Status
			}

			switch appState.Status {
			case "ALLOCATED":
				ui.Say("Application has been allocated successfully!")

				// Get the application resource
				resource, err := client.GetApplicationResource(application.UID)
				if err != nil {
					ui.Error(fmt.Sprintf("Failed to get application resource: %v", err))
					state.Put("error", fmt.Errorf("failed to get application resource: %v", err))
					return multistep.ActionHalt
				}

				if resource == nil {
					ui.Say("Application resource not ready yet, continuing to wait...")
					continue
				}

				ui.Say(fmt.Sprintf("Application resource ready (UID: %s, IP: %s)",
					resource.UID, resource.IPAddr))

				// Store the resource for other steps
				state.Put("application_resource", resource)

				// Update generated data
				generatedData := state.Get("generated_data").(map[string]any)
				generatedData["ResourceUID"] = resource.UID
				state.Put("generated_data", generatedData)

				return multistep.ActionContinue

			case "ERROR", "DEALLOCATED", "RECALLED":
				ui.Error(fmt.Sprintf("Application failed with status: %s - %s",
					appState.Status, appState.Description))
				state.Put("error", fmt.Errorf("application failed: %s", appState.Status))
				return multistep.ActionHalt

			case "NEW", "ELECTED", "DEALLOCATE":
				// These are intermediate states, continue waiting
				continue

			default:
				ui.Say(fmt.Sprintf("Unknown application status: %s", appState.Status))
				continue
			}
		}
	}
}

// Cleanup performs any necessary cleanup
func (s *StepWaitForAllocation) Cleanup(state multistep.StateBag) {
	// Nothing to clean up specifically for allocation waiting
}
