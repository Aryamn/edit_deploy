package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/util/retry"
)

//Global variable to define usage of command
var (
	editExample = `
	# --replicas = edit replicas in current namespace
	%[1]s edit-deploy <deploymentname> --replicas=<number>
	
	# --rhl = edit revison history limit in current namespace 	
	%[1]s edit-deploy <deploymentname> --rhl=<number>
	
	`
)

//Struct having all the flags arguments variable
type EditDeployOptions struct {
	configFlags *genericclioptions.ConfigFlags

	deploymentsClient v1.DeploymentInterface
	newReplicas       int32
	newRhl            int32 //Change here
	deploymentName    string

	args []string

	genericclioptions.IOStreams
}

//Function to return struct object with default value of flags
func NewEditDeploymentOptions(streams genericclioptions.IOStreams) *EditDeployOptions {
	return &EditDeployOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

//Cobra provides easy cli interface with error handling and easy extensibility(aliases, suggestions, depreciated, etc.) of cli tools
//https://cobra.dev/
func NewCmdEdit(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewEditDeploymentOptions(streams)

	cmd := &cobra.Command{
		Use:          "edit-deploy [deployment_name] [flags]",
		Short:        "View or edit current replicas",
		Example:      fmt.Sprintf(editExample, "kubectl"),
		SilenceUsage: true,
		//RunE function runs when .execute is called with error handling
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

	//Store newReplicas value in variable
	cmd.Flags().Int32Var(&o.newReplicas, "replicas", o.newReplicas, "Number of Replicas to set")
	cmd.Flags().Int32Var(&o.newRhl, "rhl", o.newRhl, "Revision History limit")
	//Add extra flags provided by user
	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

//Function to store all flags and arguments in struct
func (o *EditDeployOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	if len(args) > 0 {
		o.deploymentName = args[0]
	}

	if len(o.deploymentName) == 0 {

		return fmt.Errorf("deployment name not specified")

	}
	config, err := o.configFlags.ToRESTConfig()
	if err != nil {
		return err
	}

	//Rawconfig for extracting the current namespace
	rawconfig, err := o.configFlags.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return err
	}

	//Create a new client instance for config
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	//If namespace is provided in the flags
	userSpecifiedNamespace := *o.configFlags.Namespace

	//If not specified use default namespace
	if len(userSpecifiedNamespace) == 0 {
		userSpecifiedNamespace = rawconfig.Contexts[rawconfig.CurrentContext].Namespace

	}

	//Get deployment client in the specified namespace
	o.deploymentsClient = clientset.AppsV1().Deployments(userSpecifiedNamespace)
	result, getErr := o.deploymentsClient.Get(context.TODO(), o.deploymentName, metav1.GetOptions{})
	if getErr != nil {
		return getErr
	}

	if !isFlagPassed("replicas") {
		o.newReplicas = *result.Spec.Replicas
	}

	if !isFlagPassed("rhl") {
		o.newRhl = *result.Spec.RevisionHistoryLimit
	}

	return nil
}

//Function to validate if the arguments and flags are correct
func (o *EditDeployOptions) Validate() error {
	if len(o.args) != 1 {
		return fmt.Errorf("only one argument is allowed")
	}

	if o.newReplicas <= 0 {
		return fmt.Errorf("invalid number of replicas")
	}

	if o.newRhl < 0 {
		return fmt.Errorf("invalid value of RevisionHistoryLimit")
	}

	return nil
}

//Function to update the deployments
func (o *EditDeployOptions) Run() error {
	if len(o.deploymentName) > 0 && o.newReplicas > 0 {
		//Returns a REST client config from the flags
		config, err := o.configFlags.ToRESTConfig()
		if err != nil {
			return err
		}
		println(config)

		//Rawconfig for extracting the current namespace
		rawconfig, err := o.configFlags.ToRawKubeConfigLoader().RawConfig()
		if err != nil {
			return err
		}

		//Create a new client instance for config
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return err
		}

		// fmt.Printf("current namespace = %s\n", rawconfig.Contexts[rawconfig.CurrentContext].Namespace)
		// fmt.Printf("config flags namespace = %s\n", *o.configFlags.Namespace)
		// fmt.Printf("Default Namespace = %s\n", apiv1.NamespaceDefault)
		//logic for User specified namespace
		//Ask if the current context can be NULL or not

		//If namespace is provided in the flags
		userSpecifiedNamespace := *o.configFlags.Namespace

		//If not specified use default namespace
		if len(userSpecifiedNamespace) == 0 {
			userSpecifiedNamespace = rawconfig.Contexts[rawconfig.CurrentContext].Namespace

		}

		//Get deployment client in the specified namespace
		deploymentsClient := clientset.AppsV1().Deployments(userSpecifiedNamespace)

		//RetryOnConflict make an update to a resource when other code also doing change at same time
		//If conflict occurs it will wait for sometime
		// var DefaultRetry = wait.Backoff{
		// 	Steps:    5,
		// 	Duration: 10 * time.Millisecond,
		// 	Factor:   1.0,
		// 	Jitter:   0.1,
		// }
		//https://pkg.go.dev/k8s.io/apimachinery/pkg/util/wait#Backoff
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {

			//Get the specified deployment
			//passing empty context
			//Since no information required for Get like deadline, cancellation etc.

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
	flags := flag.NewFlagSet("kubectl-edit_deploy", flag.ExitOnError)
	flag.CommandLine = flags

	root := NewCmdEdit(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
