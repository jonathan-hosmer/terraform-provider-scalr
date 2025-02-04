
# Resource `scalr_provider_configuration_default`

Manage defaults of provider configurations for environments in Scalr. Create and destroy.

## Basic Usage

```hcl
resource "scalr_provider_configuration_default" "example" {
  environment_id = "env-xxxxxxxx"
  provider_configuration_id = "pcfg-xxxxxxxx"
}
```

## Argument Reference

* `environment_id` - (Required) ID of the environment, in the format `env-<RANDOM STRING>`.
* `provider_configuration_id` - (Required) ID of the provider configuration, in the format `pcfg-<RANDOM STRING>`. 

Note:
To make the provider configuration default, it must be shared with the specified environment.
See the definition of the resource [`scalr_provider_configuration`](scalr_provider_configuration.md) and attribute `environments` to learn more.
## Attribute Reference

All arguments plus:

* `id` - The ID of the provider configuration default.
  
## Import

To import provider configuration default use combined ID in the form `<environment_id>/<provider_configuration_id>` as the import ID. For example:

```shell

terraform import scalr_provider_configuration_default.example env-xxxxxxxx/pcfg-xxxxxxxx

```
