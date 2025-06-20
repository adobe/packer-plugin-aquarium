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

// StepSetupSSH sets up SSH connectivity using ProxySSH
type StepSetupSSH struct {
	Config     *Config
	HTTPClient *http.Client
}

// Run executes the step to setup SSH connectivity
// Run executes the step to setup SSH connectivity
func (s *StepSetupSSH) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	client := state.Get("api_client").(*APIClient)
	resource := state.Get("application_resource").(*ApplicationResource)

	ui.Say("Setting up SSH connectivity...")

	// Get SSH access credentials
	access, err := client.GetApplicationResourceAccess(resource.UID)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to get SSH access credentials: %v", err))
		state.Put("error", fmt.Errorf("failed to get SSH access: %v", err))
		return multistep.ActionHalt
	}

	ui.Say("SSH access credentials retrieved successfully")

	// Parse the SSH address
	sshHost, sshPort, err := ParseSSHAddress(access.Address)
	if err != nil {
		ui.Say(fmt.Sprintf("Unable to parse SSH address in response %q: %v", access.Address, err))
		sshHost = s.Config.Communicator.SSHHost
		sshPort = s.Config.Communicator.SSHPort
		ui.Say(fmt.Sprintf("Falling back to communicator defaults: %s:%d", sshHost, sshPort))
	}

	ui.Say(fmt.Sprintf("SSH endpoint: %s:%d", sshHost, sshPort))

	// Configure SSH settings based on what's available
	if access.Username != "" {
		s.Config.Communicator.SSHUsername = access.Username
		ui.Say(fmt.Sprintf("SSH username: %s", access.Username))
	}

	if access.Password != "" {
		s.Config.Communicator.SSHPassword = access.Password
		ui.Say("SSH password provided")
	}

	if access.Key != "" {
		s.Config.Communicator.SSHPrivateKey = []byte(access.Key)
		ui.Say("SSH private key provided")
	}

	// Set SSH port
	s.Config.Communicator.SSHPort = sshPort

	// Store SSH connection details in state
	state.Put("ssh_host", sshHost)
	state.Put("ssh_port", sshPort)
	state.Put("ssh_username", access.Username)
	state.Put("ssh_access", access)

	// Update generated data
	generatedData := state.Get("generated_data").(map[string]any)
	generatedData["SSHHost"] = sshHost
	generatedData["SSHPort"] = strconv.Itoa(sshPort)
	state.Put("generated_data", generatedData)

	ui.Say("SSH connectivity setup completed successfully")
	return multistep.ActionContinue
}

// Cleanup performs any necessary cleanup
func (s *StepSetupSSH) Cleanup(state multistep.StateBag) {
	// Nothing to clean up specifically for SSH setup
}
