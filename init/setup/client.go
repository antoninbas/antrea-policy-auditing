package client_setup

import (
	"k8s.io/client-go/rest"
	"os"
	"path/filepath"
	"context"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	networking "k8s.io/api/networking/v1"
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

func (k *Kubernetes) GetK8sPolicies() (*networking.NetworkPolicyList, error) {
	l, err := k.ClientSet.NetworkingV1().NetworkPolicies("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to list network policies")
	}
	return l, nil
}