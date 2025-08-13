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

	aquariumv2 "github.com/adobe/aquarium-fish/lib/rpc/proto/aquarium/v2"
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
	resource := state.Get("application_resource").(*aquariumv2.ApplicationResource)

	ui.Say("Setting up SSH connectivity...")

	// Get SSH access credentials
	access, err := client.GetApplicationResourceAccess(ctx, resource.GetUid())
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to get SSH access credentials: %v", err))
		state.Put("error", fmt.Errorf("failed to get SSH access: %v", err))
		return multistep.ActionHalt
	}

	ui.Say("SSH access credentials retrieved successfully")

	// Parse the SSH address
	sshHost, sshPort, err := ParseSSHAddress(access.GetAddress())
	if err != nil {
		ui.Say(fmt.Sprintf("Unable to parse SSH address in response %q: %v", access.GetAddress(), err))
		sshHost = s.Config.Communicator.SSHHost
		sshPort = s.Config.Communicator.SSHPort
		ui.Say(fmt.Sprintf("Falling back to communicator defaults: %s:%d", sshHost, sshPort))
	}

	ui.Say(fmt.Sprintf("SSH endpoint: %s:%d", sshHost, sshPort))

	// Configure SSH settings based on what's available
	if access.GetUsername() != "" {
		s.Config.Communicator.SSHUsername = access.GetUsername()
		ui.Say(fmt.Sprintf("SSH username: %s", access.GetUsername()))
	}

	if access.GetPassword() != "" {
		s.Config.Communicator.SSHPassword = access.GetPassword()
		ui.Say(fmt.Sprintf("SSH password provided: %s", access.GetPassword()))
		ui.Say(fmt.Sprintf("You can connect to the Resource by: ssh -p %d %s@%s", sshPort, access.GetUsername(), sshHost))
	}

	if access.GetKey() != "" {
		s.Config.Communicator.SSHPrivateKey = []byte(access.GetKey())
		ui.Say("SSH private key provided")
	}

	// Set SSH port
	s.Config.Communicator.SSHPort = sshPort

	// Store SSH connection details in state
	state.Put("ssh_host", sshHost)
	state.Put("ssh_port", sshPort)
	state.Put("ssh_username", access.GetUsername())
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
