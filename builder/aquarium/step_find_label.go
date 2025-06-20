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
	"strconv"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepFindLabel finds and validates the specified label
type StepFindLabel struct {
	Config     *Config
	HTTPClient *http.Client
}

// Run executes the step to find the label
func (s *StepFindLabel) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	client := state.Get("api_client").(*APIClient)

	ui.Say(fmt.Sprintf("Looking for label '%s'...", s.Config.LabelName))

	var version string
	if s.Config.LabelVersion != "" {
		version = s.Config.LabelVersion
		ui.Say(fmt.Sprintf("Searching for specific version: %s", version))
	} else {
		version = "last" // Get the latest version
		ui.Say("No version specified, will use the latest version")
	}

	// Get labels filtered by name and version
	labels, err := client.GetLabels(s.Config.LabelName, version)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to retrieve labels: %v", err))
		state.Put("error", fmt.Errorf("label retrieval failed: %v", err))
		return multistep.ActionHalt
	}

	if len(labels) == 0 {
		ui.Error(fmt.Sprintf("No labels found with name '%s'", s.Config.LabelName))
		state.Put("error", fmt.Errorf("label not found: %s", s.Config.LabelName))
		return multistep.ActionHalt
	}

	// If no specific version was requested, find the latest version
	var selectedLabel *Label
	if s.Config.LabelVersion == "" {
		maxVersion := -1
		for i, label := range labels {
			if label.Version > maxVersion {
				maxVersion = label.Version
				selectedLabel = &labels[i]
			}
		}
	} else {
		// Look for the specific version
		requestedVersion, err := strconv.Atoi(s.Config.LabelVersion)
		if err != nil {
			ui.Error(fmt.Sprintf("Invalid version format '%s': %v", s.Config.LabelVersion, err))
			state.Put("error", fmt.Errorf("invalid version format: %v", err))
			return multistep.ActionHalt
		}

		for i, label := range labels {
			if label.Version == requestedVersion {
				selectedLabel = &labels[i]
				break
			}
		}

		if selectedLabel == nil {
			ui.Error(fmt.Sprintf("Label '%s' version %d not found", s.Config.LabelName, requestedVersion))
			state.Put("error", fmt.Errorf("label version not found"))
			return multistep.ActionHalt
		}
	}

	if selectedLabel == nil {
		ui.Error(fmt.Sprintf("No suitable label found for '%s'", s.Config.LabelName))
		state.Put("error", fmt.Errorf("no suitable label found"))
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Found label '%s' version %d (UID: %s)",
		selectedLabel.Name, selectedLabel.Version, selectedLabel.UID))

	// Validate that the label has at least one definition
	if len(selectedLabel.Definitions) == 0 {
		ui.Error("Selected label has no definitions")
		state.Put("error", fmt.Errorf("label has no definitions"))
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Label has %d definition(s) available", len(selectedLabel.Definitions)))

	// Store the selected label for other steps
	state.Put("selected_label", selectedLabel)

	return multistep.ActionContinue
}

// Cleanup performs any necessary cleanup
func (s *StepFindLabel) Cleanup(state multistep.StateBag) {
	// Nothing to clean up for label lookup
}
