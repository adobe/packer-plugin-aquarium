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

//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package aquarium

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
)

const BuilderId = "aquarium.builder"

type Config struct {
	common.PackerConfig `mapstructure:",squash"`

	// AquariumFish API connection settings
	Endpoint              string `mapstructure:"endpoint" required:"true"`
	Username              string `mapstructure:"username" required:"true"`
	Password              string `mapstructure:"password" required:"true"`
	InsecureSkipTLSVerify bool   `mapstructure:"insecure_skip_tls_verify"`

	// Label specification
	LabelName    string `mapstructure:"label_name" required:"true"`
	LabelVersion string `mapstructure:"label_version"`

	// Timeout and retry settings
	ConnectionTimeout string `mapstructure:"connection_timeout"`
	ConnectionRetries int    `mapstructure:"connection_retries"`
	AllocationTimeout string `mapstructure:"allocation_timeout"`

	// Additional metadata to pass to the application
	ApplicationMetadata map[string]any `mapstructure:"application_metadata"`

	// SSH communication settings
	Communicator communicator.Config `mapstructure:",squash"`

	// Deprecated field for backward compatibility
	MockOption string `mapstructure:"mock"`

	// Parsed timeout values
	connectionTimeoutDuration time.Duration
	allocationTimeoutDuration time.Duration
}

type Builder struct {
	config Config
	runner multistep.Runner
}

func (b *Builder) ConfigSpec() hcldec.ObjectSpec { return b.config.FlatMapstructure().HCL2Spec() }

func (b *Builder) Prepare(raws ...any) (generatedVars []string, warnings []string, err error) {
	err = config.Decode(&b.config, &config.DecodeOpts{
		PluginType:  "packer.builder.aquarium",
		Interpolate: true,
	}, raws...)
	if err != nil {
		return nil, nil, err
	}

	// Set default values
	if b.config.ConnectionTimeout == "" {
		b.config.ConnectionTimeout = "30m"
	}
	if b.config.ConnectionRetries <= 0 {
		b.config.ConnectionRetries = 60
	}
	if b.config.AllocationTimeout == "" {
		b.config.AllocationTimeout = "10m"
	}

	// Parse timeout durations
	b.config.connectionTimeoutDuration, err = time.ParseDuration(b.config.ConnectionTimeout)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid connection_timeout: %v", err)
	}

	b.config.allocationTimeoutDuration, err = time.ParseDuration(b.config.AllocationTimeout)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid allocation_timeout: %v", err)
	}

	// Validate required fields
	if b.config.Endpoint == "" {
		return nil, nil, fmt.Errorf("endpoint is required")
	}
	if b.config.Username == "" {
		return nil, nil, fmt.Errorf("username is required")
	}
	if b.config.Password == "" {
		return nil, nil, fmt.Errorf("password is required")
	}
	if b.config.LabelName == "" {
		return nil, nil, fmt.Errorf("label_name is required")
	}

	// Set default SSH communicator
	if b.config.Communicator.Type == "" {
		b.config.Communicator.Type = "ssh"
	}

	// Return the placeholder for the generated data that will become available to provisioners and post-processors.
	buildGeneratedData := []string{"ApplicationUID", "ResourceUID", "SSHHost", "SSHPort"}
	return buildGeneratedData, nil, nil
}

func (b *Builder) Run(ctx context.Context, ui packer.Ui, hook packer.Hook) (packer.Artifact, error) {
	// Create HTTP client
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: b.config.InsecureSkipTLSVerify,
		},
	}
	httpClient := &http.Client{Transport: tr}

	// Cleanup is the first one to make sure we did not leave anything behind
	steps := []multistep.Step{&StepCleanup{
		Config:     &b.config,
		HTTPClient: httpClient,
	}}

	// Add AquariumFish steps
	steps = append(steps,
		&StepConnectAPI{
			Config:     &b.config,
			HTTPClient: httpClient,
		},
		&StepFindLabel{
			Config:     &b.config,
			HTTPClient: httpClient,
		},
		&StepCreateApplication{
			Config:     &b.config,
			HTTPClient: httpClient,
		},
		&StepWaitForAllocation{
			Config:     &b.config,
			HTTPClient: httpClient,
		},
		&StepSetupSSH{
			Config:     &b.config,
			HTTPClient: httpClient,
		},
		&communicator.StepConnectSSH{
			Config:    &b.config.Communicator,
			Host:      commFunc(host),
			SSHConfig: b.config.Communicator.SSHConfigFunc(),
		},
		new(commonsteps.StepProvision),
		&StepCreateImage{
			Config:     &b.config,
			HTTPClient: httpClient,
		},
	)

	// Setup the state bag and initial state for the steps
	state := new(multistep.BasicStateBag)
	state.Put("hook", hook)
	state.Put("ui", ui)
	state.Put("config", &b.config)

	// Set the value of the generated data that will become available to provisioners.
	state.Put("generated_data", map[string]any{})

	// Run!
	b.runner = commonsteps.NewRunner(steps, b.config.PackerConfig, ui)
	b.runner.Run(ctx, state)

	// If there was an error, return that
	if err, ok := state.GetOk("error"); ok {
		return nil, err.(error)
	}

	// Get the generated data
	generatedData := state.Get("generated_data").(map[string]any)

	artifact := &Artifact{
		// Add the builder generated data to the artifact StateData so that post-processors
		// can access them.
		StateData: map[string]any{"generated_data": generatedData},
	}
	return artifact, nil
}

// commFunc returns the host for SSH communication
func commFunc(host func(multistep.StateBag) (string, error)) func(multistep.StateBag) (string, error) {
	return host
}

// host returns the SSH host from the state
func host(state multistep.StateBag) (string, error) {
	sshHost, ok := state.GetOk("ssh_host")
	if !ok {
		return "", fmt.Errorf("ssh_host not found in state")
	}
	return sshHost.(string), nil
}
