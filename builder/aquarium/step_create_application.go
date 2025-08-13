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

	aquariumv2 "github.com/adobe/aquarium-fish/lib/rpc/proto/aquarium/v2"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"google.golang.org/protobuf/types/known/structpb"
)

// StepCreateApplication creates an application in AquariumFish
type StepCreateApplication struct {
	Config     *Config
	HTTPClient *http.Client
}

// Run executes the step to create an application
func (s *StepCreateApplication) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	client := state.Get("api_client").(*APIClient)
	selectedLabel := state.Get("selected_label").(*aquariumv2.Label)

	ui.Say("Creating application...")

	// Prepare application metadata
	metadata := make(map[string]any)

	// Add any user-provided metadata
	if s.Config.ApplicationMetadata != nil {
		for k, v := range s.Config.ApplicationMetadata {
			metadata[k] = v
		}
	}

	// Add packer-specific metadata
	metadata["PACKER_BUILD"] = "true"
	metadata["PACKER_BUILDER"] = "aquarium"
	metadata["PACKER_BUILD_TIME"] = time.Now().Format(time.RFC3339)

	// Create the application
	metaStruct, _ := structpb.NewStruct(metadata)
	app := &aquariumv2.Application{
		LabelUid: selectedLabel.GetUid(),
		Metadata: metaStruct,
	}

	createdApp, err := client.CreateApplication(ctx, app)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to create application: %v", err))
		state.Put("error", fmt.Errorf("application creation failed: %v", err))
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Application created successfully (UID: %s)", createdApp.GetUid()))

	// Store the created application for other steps
	state.Put("application", createdApp)

	// Update generated data
	generatedData := state.Get("generated_data").(map[string]any)
	generatedData["ApplicationUID"] = createdApp.GetUid()
	state.Put("generated_data", generatedData)

	return multistep.ActionContinue
}

// Cleanup performs any necessary cleanup
func (s *StepCreateApplication) Cleanup(state multistep.StateBag) {
	// The application cleanup will be handled by StepCleanup
}
