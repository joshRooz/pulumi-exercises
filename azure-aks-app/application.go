package main

import (
	"errors"

	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Application struct {
	pulumi.ResourceState

	Service *corev1.Service
}

type ApplicationArgs struct {
	// Required. The contents of a kubeconfig file.
	Kubeconfig pulumi.StringInput

	// Required. The naming context for the application.
	Name pulumi.StringInput

	// Optional. Container image to deploy.
	ImageName pulumi.StringInput

	// Optional. Message to display.
	Message pulumi.StringInput
}

func NewApplication(ctx *pulumi.Context, name string, args *ApplicationArgs, opts ...pulumi.ResourceOption) (*Application, error) {
	if args == nil {
		return nil, errors.New("missing one or more required application arguments")
	}

	if args.Kubeconfig == nil {
		return nil, errors.New("must be provided 'Kubeconfig'")
	}

	if args.Name == nil {
		return nil, errors.New("must be provided 'Name'")
	}

	label := pulumi.StringMap{
		"app": args.Name,
	}

	if args.ImageName == nil {
		args.ImageName = pulumi.String("joshrooz/app:v0.0.1").ToStringOutput()
	}

	if args.Message == nil {
		args.Message = pulumi.String("Pulumi").ToStringOutput()
	}

	application := &Application{}
	err := ctx.RegisterComponentResource("application-comp:app:App", name, application, opts...)
	if err != nil {
		return nil, err
	}

	k8s, err := kubernetes.NewProvider(ctx, name+"-provider", &kubernetes.ProviderArgs{
		Kubeconfig: args.Kubeconfig,
	}, pulumi.Parent(application))
	if err != nil {
		return nil, err
	}

	namespace, err := corev1.NewNamespace(ctx, name+"-ns", &corev1.NamespaceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: args.Name.ToStringOutput(),
		},
	}, pulumi.Provider(k8s), pulumi.Parent(application))
	if err != nil {
		return nil, err
	}

	_, err = appsv1.NewDeployment(ctx, name+"-deploy", &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Labels:    label,
			Namespace: namespace.ID(),
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Replicas: pulumi.Int(1),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: label,
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: label,
				},
				Spec: &corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Env: corev1.EnvVarArray{
								&corev1.EnvVarArgs{
									Name:  pulumi.String("MSG"),
									Value: args.Message,
								},
							},
							Image: args.ImageName,
							Name:  pulumi.String("application"),
						},
					},
				},
			},
		},
	}, pulumi.Provider(k8s), pulumi.Parent(application))
	if err != nil {
		return nil, err
	}

	application.Service, err = corev1.NewService(ctx, name+"-svc", &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Labels:    label,
			Namespace: namespace.ID(),
		},
		Spec: &corev1.ServiceSpecArgs{
			Ports: corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Port:       pulumi.Int(80),
					TargetPort: pulumi.Int(8080),
				},
			},
			Selector: label,
			Type:     pulumi.String("LoadBalancer"),
		},
	}, pulumi.Provider(k8s), pulumi.Parent(application))
	if err != nil {
		return nil, err
	}

	return application, nil
}

func (app *Application) GetServiceIP(ctx *pulumi.Context) pulumi.StringPtrOutput {

	service := app.Service.Status.ApplyT(func(status *corev1.ServiceStatus) *string {
		service := status.LoadBalancer.Ingress[0]
		if service.Hostname != nil {
			return service.Hostname
		}
		return service.Ip
	}).(pulumi.StringPtrOutput)

	return service
}
