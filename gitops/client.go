package gitops

import (
	"os"
	"path/filepath"
	"context"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"k8s.io/client-go/rest"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	crdclientset "antrea.io/antrea/pkg/client/clientset/versioned"
	v1alpha1 "antrea.io/antrea/pkg/apis/crd/v1alpha1"
)

type Kubernetes struct {
	PodCache  map[string][]v1.Pod
	ClientSet kubernetes.Interface
	CrdClient crdclientset.Interface
}

func NewKubernetes() (*Kubernetes, error) {
	clientSet, crdClientSet, err := Client()
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to instantiate clientsets")
	}
	return &Kubernetes{
		PodCache:  map[string][]v1.Pod{},
		ClientSet: clientSet,
		CrdClient: crdClientSet,
	}, nil
}

func Client() (*kubernetes.Clientset, *crdclientset.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := filepath.Join(
			os.Getenv("KUBECONFIG"),
		)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			klog.ErrorS(err, "unable to build config from flags, check that your KUBECONFIG file is correct!")
			return nil, nil, err
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.ErrorS(err, "unable to instantiate clientset")
		return nil, nil, err
	}
	crdclient, err := crdclientset.NewForConfig(config)
	if err != nil {
		klog.ErrorS(err, "unable to instantiate crdclientset")
		return nil, nil, err
	}
	return clientset, crdclient, nil
}

func (k *Kubernetes) GetK8sPolicies() (*networking.NetworkPolicyList, error) {
	l, err := k.ClientSet.NetworkingV1().NetworkPolicies("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (k *Kubernetes) GetAntreaPolicies() (*v1alpha1.NetworkPolicyList, error) {
	l, err := k.CrdClient.CrdV1alpha1().NetworkPolicies("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (k *Kubernetes) GetAntreaClusterPolicies() (*v1alpha1.ClusterNetworkPolicyList, error) {
	l, err := k.CrdClient.CrdV1alpha1().ClusterNetworkPolicies().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (k *Kubernetes) GetAntreaTiers() (*v1alpha1.TierList, error) {
	l, err := k.CrdClient.CrdV1alpha1().Tiers().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return l, nil
}
