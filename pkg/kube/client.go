package kube

import (
	"os"

	"k8s.io/client-go/tools/clientcmd"

	rollouts "github.com/argoproj/argo-rollouts/pkg/client/clientset/versioned/typed/rollouts/v1alpha1"
	appsclient "github.com/openshift/client-go/apps/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Clients struct exposes interfaces for kubernetes as well as openshift if available
type Clients struct {
	KubernetesClient    kubernetes.Interface
	OpenshiftAppsClient appsclient.Interface
	ArgoRolloutsClient  rollouts.ArgoprojV1alpha1Interface
}

var (
	// IsOpenshift is true if environment is Openshift, it is false if environment is Kubernetes
	IsOpenshift = isOpenshift()
	// IsArgoRollouts is true if Argo Rollout resource are in the environment
	IsArgoRollouts = isArgoRollouts()
)

// GetClients returns a `Clients` object containing both openshift and kubernetes clients with an openshift identifier
func GetClients() Clients {
	client, err := GetKubernetesClient()
	if err != nil {
		logrus.Fatalf("Unable to create Kubernetes client error = %v", err)
	}

	var appsClient *appsclient.Clientset

	if IsOpenshift {
		appsClient, err = GetOpenshiftAppsClient()
		if err != nil {
			logrus.Warnf("Unable to create Openshift Apps client error = %v", err)
		}
	}

	var argoRolloutsClient *rollouts.ArgoprojV1alpha1Client

	if IsArgoRollouts {
		argoRolloutsClient, err = GetArgoRolloutsClient()
		if err != nil {
			logrus.Warnf("Unable to create ArgoRollouts client error = %v", err)
		}
	}

	return Clients{
		KubernetesClient:    client,
		OpenshiftAppsClient: appsClient,
		ArgoRolloutsClient:  argoRolloutsClient,
	}
}

func isOpenshift() bool {
	client, err := GetKubernetesClient()
	if err != nil {
		logrus.Fatalf("Unable to create Kubernetes client error = %v", err)
	}
	_, err = client.RESTClient().Get().AbsPath("/apis/project.openshift.io").Do().Raw()
	if err == nil {
		logrus.Info("Environment: Openshift")
		return true
	}
	logrus.Info("Environment: Kubernetes")
	return false
}

func isArgoRollouts() bool {
	client, err := GetKubernetesDiscoveryClient()
	if err != nil {
		logrus.Fatalf("Unable to create Kubernetes discovery client error = %v", err)
	}
	resources, err := client.ServerResourcesForGroupVersion("argoproj.io/v1alpha1")
	if err != nil {
		logrus.Fatalf("Unable to get argoproj.io/v1alpha1 resources error = %v", err)
		return false
	}
	for _, resource := range resources.APIResources {
		if resource.Name == "rollouts" {
			return true
		}
	}
	return false
}

// GetOpenshiftAppsClient returns an Openshift Client that can query on Apps
func GetOpenshiftAppsClient() (*appsclient.Clientset, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}
	return appsclient.NewForConfig(config)
}

// GetArgoRolloutsClient returns an Openshift Client that can query on Apps
func GetArgoRolloutsClient() (*rollouts.ArgoprojV1alpha1Client, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}
	return rollouts.NewForConfig(config)
}

// GetKubernetesDiscoveryClient returns an Openshift Client that can query on Apps
func GetKubernetesDiscoveryClient() (*discovery.DiscoveryClient, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}
	return discovery.NewDiscoveryClientForConfig(config)
}

// GetKubernetesClient gets the client for k8s, if ~/.kube/config exists so get that config else incluster config
func GetKubernetesClient() (*kubernetes.Clientset, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func getConfig() (*rest.Config, error) {
	var config *rest.Config
	var err error
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("HOME") + "/.kube/config"
	}
	//If file exists so use that config settings
	if _, err := os.Stat(kubeconfigPath); err == nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, err
		}
	} else { //Use Incluster Configuration
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}
	if err != nil {
		return nil, err
	}

	return config, nil
}
