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
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepSetupSSH sets up SSH connectivity using ProxySSH
type StepSetupSSH struct {
	Config     *Config
	HTTPClient *http.Client
}

// Run executes the step to setup SSH connectivity
func (s *StepSetupSSH) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	client := state.Get("api_client").(*APIClient)
	resource := state.Get("application_resource").(*ApplicationResource)

	ui.Say("Setting up SSH connectivity...")

	// Set up timeout for initial SSH access availability check
	timeoutCtx, cancel := context.WithTimeout(ctx, s.Config.connectionTimeoutDuration)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	retryCount := 0
	maxRetries := s.Config.ConnectionRetries

	for {
		select {
		case <-timeoutCtx.Done():
			ui.Error(fmt.Sprintf("SSH access availability timeout reached (%s)", s.Config.ConnectionTimeout))
			state.Put("error", fmt.Errorf("SSH access availability timeout"))
			return multistep.ActionHalt

		case <-ticker.C:
			retryCount++
			if retryCount > maxRetries {
				ui.Error(fmt.Sprintf("Maximum SSH access availability retries reached (%d)", maxRetries))
				state.Put("error", fmt.Errorf("SSH access availability retry limit exceeded"))
				return multistep.ActionHalt
			}

			ui.Say(fmt.Sprintf("Checking SSH access availability (attempt %d/%d)...", retryCount, maxRetries))

			// Check if SSH access is available (we don't store credentials here since they're dynamic)
			access, err := client.GetApplicationResourceAccess(resource.UID)
			if err != nil {
				ui.Error(fmt.Sprintf("Failed to get SSH access credentials: %v", err))
				state.Put("error", fmt.Errorf("failed to get SSH access credentials: %v", err))
				return multistep.ActionHalt
			}

			if access == nil {
				ui.Say("SSH access not available yet, retrying...")
				continue
			}

			ui.Say("SSH access is available")

			// Parse the SSH address to get the host for connection
			sshHost, sshPort, err := ParseSSHAddress(access.Address)
			if err != nil {
				ui.Say(fmt.Sprintf("Unable to parse SSH address in response %q: %v", access.Address, err))
				sshHost = s.Config.Communicator.SSHHost
				sshPort = s.Config.Communicator.SSHPort
				ui.Say(fmt.Sprintf("Falling back to communicator defaults: %s:%d", sshHost, sshPort))
			}

			ui.Say(fmt.Sprintf("SSH endpoint: %s:%d", sshHost, sshPort))

			// Store SSH connection details in state (for the host function)
			state.Put("ssh_host", sshHost)
			state.Put("ssh_port", sshPort)

			// Update generated data
			generatedData := state.Get("generated_data").(map[string]any)
			generatedData["SSHHost"] = sshHost
			generatedData["SSHPort"] = strconv.Itoa(sshPort)
			state.Put("generated_data", generatedData)

			ui.Say("SSH connectivity setup completed successfully")
			ui.Say("Note: SSH credentials will be fetched dynamically for each connection")
			return multistep.ActionContinue
		}
	}
}

// Cleanup performs any necessary cleanup
func (s *StepSetupSSH) Cleanup(state multistep.StateBag) {
	// Nothing to clean up specifically for SSH setup
}
