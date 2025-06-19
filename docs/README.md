<!--
  Include a short overview about the plugin.

  This document is a great location for creating a table of contents for each
  of the components the plugin may provide. This document should load automatically
  when navigating to the docs directory for a plugin.

-->

### Installation

To install this plugin, copy and paste this code into your Packer configuration, then run [`packer init`](https://www.packer.io/docs/commands/init).

```hcl
packer {
  required_plugins {
    name = {
      # source represents the GitHub URI to the plugin repository without the `packer-plugin-` prefix.
      source  = "github.com/organization/name"
      version = ">=0.0.1"
    }
  }
}
```

Alternatively, you can use `packer plugins install` to manage installation of this plugin.

```sh
$ packer plugins install github.com/organization/plugin-name
```

### Components

The Aquarium plugin is created to simplify different environments access behind firewall and
utilizing the existing organization infrastructure for images building.

#### Builders

- [builder](/packer/integrations/adobe/aquarium/latest/components/builder/aquarium)
