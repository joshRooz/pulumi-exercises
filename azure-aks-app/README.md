# Overview
- Builds a docker container and pushes to DockerHub.
- Spins up an AKS cluster
- Abstracts the details of a very basic "application" deployment to k8s comprised of:
  * namespace
  * deployment
  * service
- Finally, returns the public IP of the load balancer exposing the web app



# Setup
* Add stack config
  ```sh
  pulumi stack init demo
  pulumi config set-all \
    --plaintext message="The Golden Ticket was Dwight's idea." \
    --plaintext image-name="mscarn/app:v0.0.7" \
    --plaintext registry-user="michaelGscott" \
    --secret registry-pass="<dont-show-this-to-jan>" \
    --plaintext azure-native:location="ScrantonEast2"

  ssh-keygen -t rsa -f /tmp/throwaway -N '' -C user@host \
    && pulumi config set --secret private-key < /tmp/throwaway \
    && pulumi config set public-key < /tmp/throwaway.pub \
    && rm -v /tmp/throwaway{,.pub}
  ```

* Authenticate to Azure
  ```sh
  az login --tenant <some-tenant>
  az account set -s <some-subscription>
  ```

* Deploy
  ```sh
  pulumi up [-s <stack-name>]
  ```

* Testing
  ```sh
  # running or reviewing locally? -> pull the image from DockerHub
  $(pulumi stack output dockerPullCommand)

  # verify functional app? -> curl the lb exposing aks
  curl $(pulumi stack output url) ; echo
  ```

* Clean-up
  ```sh
  pulumi destroy [-s <stack-name>] -y
  ```