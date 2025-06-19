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

integration {
  name = "Aquarium"
  description = "Aquarium builder packer plugin"
  identifier = "packer/adobe/aquarium"
  flags = [
    "hcp-ready",
    "archived",
  ]
  docs {
    process_docs = true
    readme_location = "./README.md"
    external_url = "https://github.com/adobe/packer-plugin-aquarium"
  }
  license {
    type = "Apache-2.0"
    url = "https://github.com/adobe/packer-plugin-aquarium/blob/main/LICENSE.md"
  }
  component {
    type = "builder"
    name = "Aquarium"
    slug = "name"
  }
}
