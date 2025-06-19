# Packer Plugin Aquarium

Aquarium builder plugin. It uses the existing Aquarium cluster to allocate the Application and use
it for creating new images.

## Build from source

1. Clone this GitHub repository locally.

2. Run this command from the root directory: 
```shell 
go build -ldflags="-X github.com/adobe/packer-plugin-aquarium/version.VersionPrerelease=dev" -o packer-plugin-aquarium
```

3. After you successfully compile, the `packer-plugin-aquarium` plugin binary file is in the root directory. 

4. To install the compiled plugin, run the following command 
```shell
packer plugins install --path packer-plugin-aquarium github.com/adobe/packer-plugin-aquarium
```

### Build on *nix systems
Unix like systems with the make, sed, and grep commands installed can use the `make dev` to execute the build from source steps. 

### Build on Windows Powershell
The preferred solution for building on Windows are steps 2-4 listed above.
If you would prefer to script the building process you can use the following as a guide

```powershell
$MODULE_NAME = (Get-Content go.mod | Where-Object { $_ -match "^module"  }) -replace 'module ',''
$FQN = $MODULE_NAME -replace 'packer-plugin-',''
go build -ldflags="-X $MODULE_NAME/version.VersionPrerelease=dev" -o packer-plugin-aquarium.exe
packer plugins install --path packer-plugin-aquarium.exe $FQN
```

## Running Acceptance Tests

Make sure to install the plugin locally using the steps in [Build from source](#build-from-source).

Once everything needed is set up, run:
```
PACKER_ACC=1 go test -count 1 -v ./... -timeout=120m
```

This will run the acceptance tests for all plugins in this set.

## Registering Plugin as Packer Integration

Partner and community plugins can be hard to find if a user doesn't know what 
they are looking for. To assist with plugin discovery Packer offers an integration
portal at https://developer.hashicorp.com/packer/integrations to list known integrations 
that work with the latest release of Packer. 

Registering a plugin as an integration requires [metadata configuration](./metadata.hcl) within the plugin
repository and approval by the Packer team. To initiate the process of registering your 
plugin as a Packer integration refer to the [Developing Plugins](https://developer.hashicorp.com/packer/docs/plugins/creation#registering-plugins) page.

# Requirements

-	[packer-plugin-sdk](https://github.com/hashicorp/packer-plugin-sdk) >= v0.5.2
-	[Go](https://golang.org/doc/install) >= 1.20

## Packer Compatibility
This plugin is compatible with Packer >= v1.10.2
