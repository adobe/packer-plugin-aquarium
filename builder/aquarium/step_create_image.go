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

// StepCreateImage creates an image using the TaskImage functionality
type StepCreateImage struct {
	Config     *Config
	HTTPClient *http.Client
}

// Run executes the step to create the image
func (s *StepCreateImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	client := state.Get("api_client").(*APIClient)
	application := state.Get("application").(*Application)

	ui.Say("Creating image using TaskImage...")

	// Create the image task
	// TODO: Fix image creation - pass the name of the image to fish
	imageTask := ApplicationTask{
		ApplicationUID: application.UID,

		Task:    "TaskImage",
		When:    "DEALLOCATE",
		Options: map[string]any{},
	}

	// Create the task
	createdTask, err := client.CreateApplicationTask(application.UID, imageTask)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to create image task: %v", err))
		state.Put("error", fmt.Errorf("image task creation failed: %v", err))
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Image task created (UID: %s)", createdTask.UID))

	// Set up timeout for image creation
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Minute) // Allow more time for image creation
	defer cancel()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	ui.Say("Waiting for image creation to complete...")

	for {
		select {
		case <-timeoutCtx.Done():
			ui.Error("Image creation timeout reached")
			state.Put("error", fmt.Errorf("image creation timeout"))
			return multistep.ActionHalt

		case <-ticker.C:
			// Get current task status
			currentTask, err := client.GetApplicationTask(createdTask.UID)
			if err != nil {
				ui.Error(fmt.Sprintf("Failed to get task status: %v", err))
				state.Put("error", fmt.Errorf("failed to get task status: %v", err))
				return multistep.ActionHalt
			}

			// Check if task has results (indicating completion)
			if currentTask.Result != nil && len(currentTask.Result) > 0 {
				ui.Say("Image creation completed!")

				// Check for success/failure in results
				if status, exists := currentTask.Result["status"]; exists {
					if status == "success" || status == "completed" {
						ui.Say("Image created successfully")

						// Check for image information in results
						if imageInfo, exists := currentTask.Result["image"]; exists {
							ui.Say(fmt.Sprintf("Image information: %v", imageInfo))
						}

						if imagePath, exists := currentTask.Result["image_path"]; exists {
							ui.Say(fmt.Sprintf("Image path: %s", imagePath))
						}

						// Store image task results
						state.Put("image_task", currentTask)
						state.Put("image_results", currentTask.Result)

						return multistep.ActionContinue
					} else if status == "failed" || status == "error" {
						ui.Error(fmt.Sprintf("Image creation failed: %v", currentTask.Result))
						state.Put("error", fmt.Errorf("image creation failed"))
						return multistep.ActionHalt
					}
				}

				// If no explicit status, assume success if results are present
				ui.Say("Image creation appears to have completed")
				state.Put("image_task", currentTask)
				state.Put("image_results", currentTask.Result)
				return multistep.ActionContinue
			}

			ui.Message("Image creation still in progress...")
		}
	}
}

// Cleanup performs any necessary cleanup
func (s *StepCreateImage) Cleanup(state multistep.StateBag) {
	// Nothing specific to clean up for image creation
	// The image task will be managed by the AquariumFish system
}
