# Copyright 2025 Adobe. All rights reserved.
# This file is licensed to you under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License. You may obtain a copy
# of the License at http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software distributed under
# the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
# OF ANY KIND, either express or implied. See the License for the specific language
# governing permissions and limitations under the License.

# Author: Sergei Parshev (@sparshev)

# Variables for configuration
variable "aquarium_endpoint" {
  description = "AquariumFish API endpoint"
  type        = string
  default     = env("AQUARIUM_ENDPOINT")
}

variable "aquarium_username" {
  description = "AquariumFish API username"
  type        = string
  default     = env("AQUARIUM_USERNAME")
}

variable "aquarium_password" {
  description = "AquariumFish API password"
  type        = string
  default     = env("AQUARIUM_PASSWORD")
  sensitive   = true
}

variable "label_name" {
  description = "Name of the label to use"
  type        = string
  default     = "ubuntu-20.04"
}

variable "label_version" {
  description = "Version of the label to use (optional)"
  type        = string
  default     = ""
}

packer {
  required_plugins {
    aquarium = {
      version = ">=v0.1.0"
      source  = "github.com/adobe/aquarium"
    }
  }
}

source "aquarium-builder" "example" {
  # AquariumFish connection settings
  endpoint               = var.aquarium_endpoint
  username               = var.aquarium_username
  password               = var.aquarium_password
  insecure_skip_tls_verify = false
  
  # Label configuration
  label_name    = var.label_name
  label_version = var.label_version
  
  # Timeout settings
  connection_timeout = "30m"
  connection_retries = 60
  allocation_timeout = "10m"
  
  # Additional metadata to pass to the application
  application_metadata = {
    BUILD_NAME = "packer-aquarium-example"
    BUILD_TIME = formatdate("YYYY-MM-DD hh:mm:ss ZZZ", timestamp())
  }
  
  # SSH communicator settings
  communicator = "ssh"
}

build {
  sources = [
    "source.aquarium-builder.example"
  ]

  # Install some packages
  provisioner "shell" {
    inline = [
      "sudo apt-get update",
      "sudo apt-get install -y curl wget git",
      "echo 'Packer build completed on AquariumFish!' > /tmp/packer-build.txt"
    ]
  }

  # Create a simple file to show the build worked
  provisioner "shell" {
    inline = [
      "echo 'Build metadata:' > /tmp/build-info.txt",
      "echo 'Application UID: ${build.ApplicationUID}' >> /tmp/build-info.txt",
      "echo 'Resource UID: ${build.ResourceUID}' >> /tmp/build-info.txt",
      "echo 'SSH Host: ${build.SSHHost}:${build.SSHPort}' >> /tmp/build-info.txt",
      "date >> /tmp/build-info.txt"
    ]
  }
}
