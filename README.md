# Terraform Provider pgrmongodb

## Run provider tests

```shell
$ cd /path/to/terraform-provider-pgrmongodb
$ TF_ACC=1 TF_LOG=INFO TF_LOG_PATH=tflog go test -timeout 99999s -v ./...

$ # for a specific test only (example using TestAccPGRMongoDBAtlasContainers)
$ TF_ACC=1 TF_LOG=INFO TF_LOG_PATH=tflog go test -timeout 99999s -run '^TestAccPGRMongoDBAppFunction$' -v ./...
```

## Build provider

Run the following command to build and deploy the provider to your workstation.

```shell
$ make
```
