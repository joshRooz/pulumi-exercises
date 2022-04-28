package main

import (
	"encoding/base64"
	"os"

	"github.com/pulumi/pulumi-azure-native/sdk/go/azure/containerservice"
	"github.com/pulumi/pulumi-azure-native/sdk/go/azure/resources"
	"github.com/pulumi/pulumi-docker/sdk/v3/go/docker"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		// get stack configuration
		config := config.New(ctx, "")
		registryServer := config.Require("registry-server")
		registryUser := config.Require("registry-user")
		registryPass := config.RequireSecret("registry-pass")
		imageName := config.Require("image-name")
		pubKey := config.Require("public-key")

		msg := config.Get("message")
		if msg == "" {
			msg = "🐡"
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		// build container and push to the registry which is configurable
		// for each stack through config. hooks/authZ between {a,g,e}ks
		// managed service and {a,g,e}cr registry not accounted for
		image, err := docker.NewImage(ctx, "app", &docker.ImageArgs{
			ImageName: pulumi.String(imageName),
			Build: &docker.DockerBuildArgs{
				Context: pulumi.Sprintf("%v/%v", cwd, "app"),
				Args:    pulumi.StringMap{"MSG": pulumi.String("Build Message 👻")},
			},
			Registry: &docker.ImageRegistryArgs{
				Server:   pulumi.String(registryServer),
				Username: pulumi.String(registryUser),
				Password: registryPass.ToStringOutput(),
			},
		})
		if err != nil {
			return err
		}

		pet, err := random.NewRandomPet(ctx, "aks-name", &random.RandomPetArgs{
			Length:    pulumi.Int(2),
			Prefix:    pulumi.String("aks"),
			Separator: pulumi.String("-"),
		})
		if err != nil {
			return err
		}

		rg, err := resources.NewResourceGroup(ctx, "rg", &resources.ResourceGroupArgs{
			ResourceGroupName: pulumi.Sprintf("rg-%s", pet.ID()),
		})
		if err != nil {
			return err
		}

		aks, err := containerservice.NewManagedCluster(ctx, "aks", &containerservice.ManagedClusterArgs{
			AgentPoolProfiles: containerservice.ManagedClusterAgentPoolProfileArray{
				&containerservice.ManagedClusterAgentPoolProfileArgs{
					Count:              pulumi.Int(1),
					EnableAutoScaling:  pulumi.Bool(false),
					EnableNodePublicIP: pulumi.Bool(true),
					Mode:               pulumi.String("System"),
					Name:               pulumi.String("default"),
					OsType:             pulumi.String("Linux"),
					Type:               pulumi.String("VirtualMachineScaleSets"),
					VmSize:             pulumi.String("Standard_B2s"),
				},
			},
			DnsPrefix:         pet.ID(),
			EnableRBAC:        pulumi.Bool(true),
			Identity:          &containerservice.ManagedClusterIdentityArgs{Type: containerservice.ResourceIdentityTypeSystemAssigned},
			KubernetesVersion: pulumi.String("1.23.5"),
			LinuxProfile: &containerservice.ContainerServiceLinuxProfileArgs{
				AdminUsername: pulumi.String("azureuser"),
				Ssh: &containerservice.ContainerServiceSshConfigurationArgs{
					PublicKeys: &containerservice.ContainerServiceSshPublicKeyArray{
						&containerservice.ContainerServiceSshPublicKeyArgs{
							KeyData: pulumi.String(pubKey),
						},
					},
				},
			},
			NetworkProfile: &containerservice.ContainerServiceNetworkProfileArgs{
				LoadBalancerProfile: &containerservice.ManagedClusterLoadBalancerProfileArgs{
					ManagedOutboundIPs: &containerservice.ManagedClusterLoadBalancerProfileManagedOutboundIPsArgs{
						Count: pulumi.Int(2),
					},
				},
				LoadBalancerSku: pulumi.String("standard"),
				OutboundType:    pulumi.String("loadBalancer"),
			},
			NodeResourceGroup: pulumi.Sprintf("%s-MC", rg.Name),
			ResourceGroupName: rg.Name,
			ResourceName:      pet.ID(),
			Sku: &containerservice.ManagedClusterSKUArgs{
				Name: pulumi.String("Basic"),
				Tier: pulumi.String("Free"),
			},
		})
		if err != nil {
			return err
		}

		// obtain cluster user credentials so we can get a kubeconfig which will be used for deployment
		creds := containerservice.ListManagedClusterUserCredentialsOutput(ctx, containerservice.ListManagedClusterUserCredentialsOutputArgs{
			ResourceGroupName: rg.Name,
			ResourceName:      aks.Name,
		})

		kubeConfig := creds.Kubeconfigs().Index(pulumi.Int(0)).Value().ApplyT(func(arg string) string {
			kubeConfig, err := base64.StdEncoding.DecodeString(arg)
			if err != nil {
				return ""
			}
			return string(kubeConfig)
		}).(pulumi.StringOutput)

		app, err := NewApplication(ctx, "k8s-app", &ApplicationArgs{
			Kubeconfig: kubeConfig,
			Name:       pulumi.String("test"),
			ImageName:  image.BaseImageName,
			Message:    pulumi.String(msg),
		})
		if err != nil {
			return err
		}

		ctx.Export("dockerPullCommand", pulumi.Sprintf("docker pull %s", image.BaseImageName))
		ctx.Export("ip", app.GetServiceIP(ctx))

		return nil
	})
}
