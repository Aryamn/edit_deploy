package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

var (
	editExample = `
	# edit replicas in current namespace
	%[1]s edit-deploy <deploymentname> --replicas=<number>`
)

type EditDeployOptions struct {
	configFlags *genericclioptions.ConfigFlags

	newReplicas    int32
	deploymentName string

	args []string

	genericclioptions.IOStreams
}

func NewEditDeploymentOptions(streams genericclioptions.IOStreams) *EditDeployOptions {
	return &EditDeployOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdEdit(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewEditDeploymentOptions(streams)

	cmd := &cobra.Command{
		Use:          "edit-deploy [deployment_name] [flags]",
		Short:        "View or edit current replicas",
		Example:      fmt.Sprintf(editExample, "kubectl"),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil

		},
	}

	cmd.Flags().Int32Var(&o.newReplicas, "replicas", o.newReplicas, "Number of Replicas to set")
	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *EditDeployOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	if len(args) > 0 {
		o.deploymentName = args[0]
	}

	if len(o.deploymentName) == 0 {

		return fmt.Errorf("deployment name not specified")

	}
	return nil
}

func (o *EditDeployOptions) Validate() error {
	if len(o.args) != 1 {
		return fmt.Errorf("only one argument is allowed")
	}

	if o.newReplicas <= 0 {
		return fmt.Errorf("invalid number of replicas")
	}
	return nil
}

func (o *EditDeployOptions) Run() error {
	if len(o.deploymentName) > 0 && o.newReplicas > 0 {
		config, err := o.configFlags.ToRESTConfig()
		if err != nil {
			return err
		}

		rawconfig, err := o.configFlags.ToRawKubeConfigLoader().RawConfig()
		if err != nil {
			return err
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return err
		}

		// fmt.Printf("current namespace = %s\n", rawconfig.Contexts[rawconfig.CurrentContext].Namespace)
		// fmt.Printf("config flags namespace = %s\n", *o.configFlags.Namespace)
		// fmt.Printf("Default Namespace = %s\n", apiv1.NamespaceDefault)
		//logic for User specified namespace
		//Ask if the current context can be NULL or not
		userSpecifiedNamespace := *o.configFlags.Namespace

		if len(userSpecifiedNamespace) == 0 {
			userSpecifiedNamespace = rawconfig.Contexts[rawconfig.CurrentContext].Namespace

		}

		deploymentsClient := clientset.AppsV1().Deployments(userSpecifiedNamespace)

		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			result, getErr := deploymentsClient.Get(context.TODO(), o.deploymentName, metav1.GetOptions{})

			if getErr != nil {
				return fmt.Errorf("failed to get latest version fo Deployment: %v", getErr)
				// return getErr
			}

			result.Spec.Replicas = &o.newReplicas
			_, updateErr := deploymentsClient.Update(context.TODO(), result, metav1.UpdateOptions{})
			return updateErr
		})

		if retryErr != nil {
			return fmt.Errorf("update failed: %v", retryErr)
			// return retryErr
		}
		fmt.Println("Updated Deployment..")
	}
	return nil
}

func main() {
	flags := pflag.NewFlagSet("kubectl-edit_deploy", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := NewCmdEdit(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
