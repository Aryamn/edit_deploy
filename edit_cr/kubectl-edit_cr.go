package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"

	v1 "k8s.io/api/rbac/v1"
	typev1 "k8s.io/client-go/kubernetes/typed/rbac/v1"

	"k8s.io/client-go/util/retry"
)

//Global variable to define usage of command
var (
	editExample = `
	#--verbs = specify operation to be mentioned seperated by ","
	#--resources = specify resources to be mentioned seperated by ","
	#--groups = specify groups that resources belongs to seperated by ","
	%[1]s edit-cr <clusterResourceName> --verbs=update,delete --resources=downloads,links --groups=data.falcon.io
	
	`
)

//Struct having all the flags arguments variable
type EditDeployOptions struct {
	configFlags *genericclioptions.ConfigFlags

	clusterRoleInterface typev1.ClusterRoleInterface
	newVerbs             string
	newApiGroups         string
	newResources         string
	clusterRoleName      string

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

//Cobra provides easy cli interface with error handling and easy extensibility(aliases, suggestions, depreciated, etc.) of cli tools
//https://cobra.dev/
func NewCmdEdit(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewEditDeploymentOptions(streams)

	cmd := &cobra.Command{
		Use:          "edit-cr [ClusterRoleName] [flags]",
		Short:        "Append rules to Specified ClusterRole",
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
	cmd.Flags().StringVar(&o.newVerbs, "verbs", o.newVerbs, "Comma seperated verb actions")
	cmd.Flags().StringVar(&o.newApiGroups, "groups", o.newApiGroups, "comma seperated api groups")
	cmd.Flags().StringVar(&o.newResources, "resources", o.newResources, "comma seperated Resources")

	//Add extra flags provided by user
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

//Function to store all flags and arguments in struct
func (o *EditDeployOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	if len(args) > 0 {
		o.clusterRoleName = args[0]
	}

	if len(o.clusterRoleName) == 0 {

		return fmt.Errorf("ClusterRole name not specified")

	}

	config, err := o.configFlags.ToRESTConfig()
	if err != nil {
		return err
	}

	//Create a new client instance for config
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	//Get ClusterRole Interface
	o.clusterRoleInterface = clientset.RbacV1().ClusterRoles()

	return nil
}

//Function to validate if the arguments and flags are correct
func (o *EditDeployOptions) Validate() error {
	if len(o.args) != 1 {
		return fmt.Errorf("only one argument is allowed")
	}

	if len(o.newVerbs) == 0 {
		return fmt.Errorf("verb feild is empty")
	}

	if len(o.newResources) == 0 {
		return fmt.Errorf("resource feild is empty")
	}

	return nil
}

//Function to update the deployments
func (o *EditDeployOptions) Run() error {

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

		//Get the specified ClusterRole
		//passing empty context
		//Since no information required for Get like deadline, cancellation etc.

		result, getErr := o.clusterRoleInterface.Get(context.TODO(), o.clusterRoleName, metav1.GetOptions{})

		if getErr != nil {
			return fmt.Errorf("failed to get latest version fo Deployment: %v", getErr)
		}

		listVerbs := strings.Split(o.newVerbs, ",")
		listResources := strings.Split(o.newResources, ",")
		listApiGroups := strings.Split(o.newApiGroups, ",")
		result.Rules = append(result.Rules, v1.PolicyRule{Verbs: listVerbs, Resources: listResources, APIGroups: listApiGroups})

		_, updateErr := o.clusterRoleInterface.Update(context.TODO(), result, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return fmt.Errorf("update failed: %v", retryErr)
	}
	fmt.Println("Updated ClusterRoles..")

	return nil
}

func main() {
	flags := flag.NewFlagSet("kubectl-edit_cr", flag.ExitOnError)
	flag.CommandLine = flags

	root := NewCmdEdit(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
