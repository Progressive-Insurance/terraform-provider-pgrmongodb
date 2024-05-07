# Terraform Provider pgrmongodb

## Run provider tests

```shell
$ cd /path/to/terraform-provider-pgrmongodb
$ TF_ACC=1 TF_LOG=INFO TF_LOG_PATH=tflog go test -timeout 99999s -v ./...

$ # for a specific test only (example using TestAccPGRMongoDBAtlasContainers)
$ TF_ACC=1 TF_LOG=INFO TF_LOG_PATH=tflog go test -timeout 99999s -run TestAccPGRMongoDBAtlasContainers -v ./...
```

## Build provider

Run the following command to build and deploy the provider to your workstation.

```shell
$ make
```

## Test sample configuration

Navigate to the `examples` directory.

```shell
$ cd examples
```

Run the following command to initialize the workspace and apply the sample configuration.

```shell
$ terraform init && terraform apply
```
