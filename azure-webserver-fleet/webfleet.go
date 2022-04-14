package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"

	"github.com/pulumi/pulumi-azure-native/sdk/go/azure/compute"
	"github.com/pulumi/pulumi-azure-native/sdk/go/azure/resources"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type WebServerFleet struct {
	pulumi.ResourceState
}

type Machine struct {
	// An optional username for the VM login. Default: azadmin
	Username pulumi.StringInput

	// An optional password for the VM password. Default: thisisntgood
	Password pulumi.StringInput

	// An optional operating system for the VM. Default: ubuntu
	OperatingSystem pulumi.StringInput

	// An optional VM size. Default: small
	Size pulumi.StringInput

	// Required. Subnet ID to deploy the VMs into.
	SubnetID pulumi.StringPtrInput

	// An optional number of VMs to create. Default: 1
	Count pulumi.Float64Input
}

type WebServerFleetArgs struct {
	Machines []Machine
}

func NewWebServerFleet(ctx *pulumi.Context, name string, args *WebServerFleetArgs, opts ...pulumi.ResourceOption) (*WebServerFleet, error) {

	// Read in our provisioning script and prepare the custom data input
	buf, err := ioutil.ReadFile("./provision.sh")
	if err != nil {
		return nil, err
	}
	customData := base64.StdEncoding.EncodeToString(buf)

	var wbSrvrFlt WebServerFleet
	err = ctx.RegisterComponentResource("wsf-go-azure-comp:websrvrflt:WebServerFleet", name, &wbSrvrFlt, opts...)
	if err != nil {
		return nil, err
	}

	for i, m := range args.Machines {

		// Create a unique id, set our defaults, and determine os.
		// im relying on the azure-native provider for all but the
		// most basic input sanization and error handling at the moment
		id := fmt.Sprintf("%s%v", name, i)
		setDefaults(&m)
		var image *compute.ImageReferenceArgs
		if m.OperatingSystem == pulumi.String("centos") {
			image = &compute.ImageReferenceArgs{
				Offer:     pulumi.String("CentOs"),
				Publisher: pulumi.String("OpenLogic"),
				Sku:       pulumi.String("7_9"),
				Version:   pulumi.String("latest"),
			}
		} else {
			image = &compute.ImageReferenceArgs{
				Offer:     pulumi.String("UbuntuServer"),
				Publisher: pulumi.String("canonical"),
				Sku:       pulumi.String("18.04-LTS"),
				Version:   pulumi.String("latest"),
			}
		}

		rg, err := resources.NewResourceGroup(ctx, "rg-"+id, nil, pulumi.Parent(&wbSrvrFlt))
		if err != nil {
			return nil, err
		}

		_, err = compute.NewVirtualMachineScaleSet(ctx, "vmss-"+id, &compute.VirtualMachineScaleSetArgs{
			Overprovision:     pulumi.Bool(true),
			ResourceGroupName: rg.Name,
			Sku: &compute.SkuArgs{
				Capacity: m.Count,
				Name:     m.Size,
			},
			UpgradePolicy: &compute.UpgradePolicyArgs{
				Mode: compute.UpgradeMode("Manual"),
			},
			VirtualMachineProfile: &compute.VirtualMachineScaleSetVMProfileArgs{
				NetworkProfile: &compute.VirtualMachineScaleSetNetworkProfileArgs{
					NetworkInterfaceConfigurations: compute.VirtualMachineScaleSetNetworkConfigurationArray{
						&compute.VirtualMachineScaleSetNetworkConfigurationArgs{
							IpConfigurations: compute.VirtualMachineScaleSetIPConfigurationArray{
								&compute.VirtualMachineScaleSetIPConfigurationArgs{
									LoadBalancerBackendAddressPools: compute.SubResourceArray{},
									LoadBalancerInboundNatPools:     compute.SubResourceArray{},
									Name:                            pulumi.Sprintf("ipcfg-vmss-%s", id),
									PublicIPAddressConfiguration: &compute.VirtualMachineScaleSetPublicIPAddressConfigurationArgs{
										Name: pulumi.Sprintf("pip-vmss-%s", id),
									},
									Subnet: &compute.ApiEntityReferenceArgs{Id: m.SubnetID},
								},
							},
							Name:                 pulumi.Sprintf("nic-vmss-%s", id),
							NetworkSecurityGroup: nil,
							Primary:              pulumi.Bool(true),
						},
					},
				},
				OsProfile: &compute.VirtualMachineScaleSetOSProfileArgs{
					AdminPassword:      m.Password,
					AdminUsername:      m.Username,
					ComputerNamePrefix: m.OperatingSystem,
					LinuxConfiguration: &compute.LinuxConfigurationArgs{
						DisablePasswordAuthentication: pulumi.BoolPtr(false),
					},
					CustomData: pulumi.String(customData),
				},
				StorageProfile: &compute.VirtualMachineScaleSetStorageProfileArgs{
					ImageReference: image,
					OsDisk: &compute.VirtualMachineScaleSetOSDiskArgs{
						Caching:      compute.CachingTypes("ReadWrite"),
						CreateOption: pulumi.String("FromImage"),
						ManagedDisk: &compute.VirtualMachineScaleSetManagedDiskParametersArgs{
							StorageAccountType: pulumi.String("Standard_LRS"),
						},
					},
				},
			},
		}, pulumi.Parent(&wbSrvrFlt))
		if err != nil {
			return nil, err
		}
	}

	return &wbSrvrFlt, nil
}

// Cleaning up parts of this hacky
// function would be the next optimization
// ie converting to a NewMachine method
func setDefaults(m *Machine) {
	if m.Username == nil {
		m.Username = pulumi.String("azadmin")
	}

	if m.Password == nil {
		m.Password = pulumi.String("thIs!sntg0od")
	}

	if m.OperatingSystem != pulumi.String("ubuntu") && m.OperatingSystem != pulumi.String("centos") {
		m.OperatingSystem = pulumi.String("ubuntu")
	}

	switch m.Size {
	case pulumi.String("medium"):
		m.Size = pulumi.String("Standard_B1s")
	case pulumi.String("large"):
		m.Size = pulumi.String("Standard_B2s")
	default:
		m.Size = pulumi.String("Standard_B1ls")
	}

	if m.Count == nil {
		m.Count = pulumi.Float64(1)
	}
}
