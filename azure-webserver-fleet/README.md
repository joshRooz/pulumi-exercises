# README

Create the Pulumi Project
  ```sh
  mkdir -pv azure-webserver-fleet && cd azure-webserver-fleet
  pulumi new azure-go
  ```

Write the Program
- edit/write `main.go`
- write `webfleet.go`

Add Config
```sh
pulumi config set vm-username adm-user1
pulumi config set --secret vm-password <some-secret>
```

Deploy to Azure
```sh
az login --tenant <some-tenant>
az account set -s <some-subscription>

pulumi up -s <stack-name>
```

Test - *add exports to Pulumi instead of using az cli*
```sh
az vmss list --query "[].[name,resourceGroup]" -o tsv

az vmss list-instance-public-ips \
  -g <resource-group> \
  -n <vmss-name> \
  --query "[].[name,ipAddress]" \
  -o tsv

curl http://<ip-address>
```

Destroy
```sh
pulumi destroy -s <stack-name>
```