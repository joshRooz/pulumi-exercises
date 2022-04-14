package main

import (
	"github.com/pulumi/pulumi-azure-native/sdk/go/azure/network"
	"github.com/pulumi/pulumi-azure-native/sdk/go/azure/resources"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		conf := config.New(ctx, "")
		username := conf.Require("vm-username")
		password := conf.Require("vm-password")

		// Create an Azure Resource Group for the 'platform' network
		resourceGroup, err := resources.NewResourceGroup(ctx, "rg-platform", nil)
		if err != nil {
			return err
		}

		// Creates a Virtual Network. This might be part of a different
		// project/stack depending on how customer operates and/or environment
		vn, err := network.NewVirtualNetwork(ctx, "vn-platform", &network.VirtualNetworkArgs{
			AddressSpace: &network.AddressSpaceArgs{
				AddressPrefixes: pulumi.StringArray{
					pulumi.String("10.0.0.0/20"),
				},
			},
			ResourceGroupName: resourceGroup.Name,
			Subnets: network.SubnetTypeArray{
				&network.SubnetTypeArgs{
					AddressPrefix: pulumi.String("10.0.0.0/24"),
					Name:          pulumi.String("sn-01"),
				},
				&network.SubnetTypeArgs{
					AddressPrefix: pulumi.String("10.0.1.0/24"),
					Name:          pulumi.String("sn-02"),
				},
			},
		})
		if err != nil {
			return err
		}

		// Create a new web server fleet using virtual machine scale sets in Azure
		// encapsulates centos and ubuntu provisioning for small, medium, and large.
		// each vmss instance is spun up in its own resource group which could serve
		// as a logical control plane boundary
		_, err = NewWebServerFleet(ctx, "ws-fleet", &WebServerFleetArgs{
			Machines: []Machine{
				{
					Username:        pulumi.String(username),
					Password:        pulumi.String(password),
					OperatingSystem: pulumi.String("ubuntu"),
					Size:            pulumi.String("small"),
					SubnetID:        vn.Subnets.Index(pulumi.Int(0)).Id(),
					Count:           pulumi.Float64(2),
				},
				{
					Username:        pulumi.String(username),
					Password:        pulumi.String(password),
					OperatingSystem: pulumi.String("centos"),
					Size:            pulumi.String("medium"),
					SubnetID:        vn.Subnets.Index(pulumi.Int(1)).Id(),
					Count:           pulumi.Float64(1),
				},
			},
		})
		if err != nil {
			return err
		}

		return nil
	})
}
