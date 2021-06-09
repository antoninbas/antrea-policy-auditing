package main

import (
	"fmt"
	"k8s.io/client-go/rest"
	"os"
	"path/filepath"
	"context"

	"github.com/pkg/errors"
	"github.com/ghodss/yaml"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Kubernetes struct {
	podCache  map[string][]v1.Pod
	ClientSet *kubernetes.Clientset
}

func NewKubernetes() (*Kubernetes, error) {
	clientSet, err := Client()
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to instantiate kube client")
	}
	return &Kubernetes{
		podCache:  map[string][]v1.Pod{},
		ClientSet: clientSet,
	}, nil
}

func Client() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := filepath.Join(
			os.Getenv("KUBECONFIG"), //TODO: Update path based on where this file is run from
		)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, errors.WithMessagef(err, "unable to build config from flags, check that your KUBECONFIG file is correct !")
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to instantiate clientset")
	}
	return clientset, nil
}

func (k *Kubernetes) ListK8sPolicies() error {
	l, err := k.ClientSet.NetworkingV1().NetworkPolicies("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrapf(err, "unable to list network policies")
	}
	for _, np := range l.Items {
		y, err := yaml.JSONToYAML([]byte(np.Annotations["kubectl.kubernetes.io/last-applied-configuration"]))
		if err != nil {
			return errors.Wrapf(err, "unable to convert network policy object")
		}
		fmt.Println(string(y))
	}
	return nil
}

func main() {
	k8s, err := NewKubernetes()
	if err != nil {
		fmt.Printf("something went wrong")
	}
	k8s.ListK8sPolicies()
}