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

packer {
  required_plugins {
    aquarium = {
      version = ">=v0.1.0"
      source  = "github.com/adobe/packer-plugin-aquarium"
    }
  }
}

source "aquarium-builder" "foo-example" {
  mock = local.foo
}

source "aquarium-builder" "bar-example" {
  mock = local.bar
}

build {
  sources = [
    "source.aquarium-builder.foo-example",
  ]

  source "source.aquarium-builder.bar-example" {
    name = "bar"
  }

  provisioner "aquarium-provisioner" {
    only = ["aquarium-builder.foo-example"]
    mock = "foo: ${local.foo}"
  }

  provisioner "aquarium-provisioner" {
    only = ["aquarium-builder.bar"]
    mock = "bar: ${local.bar}"
  }

  post-processor "aquarium-post-processor" {
    mock = "post-processor mock-config"
  }
}
