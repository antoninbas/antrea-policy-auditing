package gitops

import (
	"context"
	"os"

	"antrea.io/antrea/pkg/apis/crd/v1alpha1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8sClient struct {
	client.Client
}

func NewKubernetes() (*K8sClient, error) {
	var config *rest.Config
	kubeconfig, hasIt := os.LookupEnv("KUBECONFIG")
	if !hasIt {
		kubeconfig = clientcmd.RecommendedHomeFile
	}
	if _, err := os.Stat(kubeconfig); kubeconfig == clientcmd.RecommendedHomeFile && os.IsNotExist(err) {
		config, err = rest.InClusterConfig()
		if err != nil {
			klog.ErrorS(err, "unable to create InClusterConfig")
			return nil, err
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			klog.ErrorS(err, "unable to build config from flags, check that your KUBECONFIG file is correct!")
			return nil, err
		}
	}
	scheme := runtime.NewScheme()
	RegisterTypes(scheme)
	client, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		klog.ErrorS(err, "unable to instantiate new generic client")
		return nil, err
	}
	return &K8sClient{client}, nil
}

func RegisterTypes(scheme *runtime.Scheme) {
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "networking.k8s.io",
		Version: "v1",
		Kind:    "NetworkPolicy"},
		&networking.NetworkPolicy{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "networking.k8s.io",
		Version: "v1",
		Kind:    "NetworkPolicyList"},
		&networking.NetworkPolicyList{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "networking.k8s.io",
		Version: "v1",
		Kind:    "ListOptions"},
		&metav1.ListOptions{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "NetworkPolicy"},
		&v1alpha1.NetworkPolicy{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "NetworkPolicyList"},
		&v1alpha1.NetworkPolicyList{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "ClusterNetworkPolicy"},
		&v1alpha1.ClusterNetworkPolicy{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "ClusterNetworkPolicyList"},
		&v1alpha1.ClusterNetworkPolicyList{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "Tier"},
		&v1alpha1.Tier{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "TierList"},
		&v1alpha1.TierList{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "ListOptions"},
		&metav1.ListOptions{})
}

func (k *K8sClient) GetResource(resource *unstructured.Unstructured, namespace string, name string) (*unstructured.Unstructured, error) {
	err := k.Get(context.TODO(), client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, resource)
	if err != nil {
		klog.Errorf("unable to get resource %s/%s", namespace, name)
		return nil, err
	}
	return resource, nil
}

func (k *K8sClient) ListResource(resourceList *unstructured.UnstructuredList) (*unstructured.UnstructuredList, error) {
	err := k.List(context.TODO(), resourceList)
	if err != nil {
		klog.ErrorS(err, "unable to list resource")
		return nil, err
	}
	return resourceList, nil
}

func (k *K8sClient) CreateOrUpdateResource(resource *unstructured.Unstructured) error {
	if err := k.Create(context.TODO(), resource); err == nil {
		klog.V(2).Infof("created resource %s", resource.GetName())
		return nil
	} else if errors.IsAlreadyExists(err) {
		klog.Infof("resource %s already exists, trying update instead", resource.GetName())
		oldResource := &unstructured.Unstructured{}
		oldResource.SetGroupVersionKind(resource.GroupVersionKind())
		_ = k.Get(context.TODO(), client.ObjectKey{
			Namespace: resource.GetNamespace(),
			Name:      resource.GetName(),
		}, oldResource)
		resource.SetResourceVersion(oldResource.GetResourceVersion())
		if err := k.Update(context.TODO(), resource); err != nil {
			klog.Errorf("unable to update resource %s", resource.GetName())
			return err
		}
		klog.V(2).Infof("updated resource %s", resource.GetName())
		return nil
	} else {
		klog.ErrorS(err, "error while creating resource", "resourceName", resource.GetName())
		return err
	}
}

func (k *K8sClient) DeleteResource(resource *unstructured.Unstructured) error {
	err := k.Delete(context.TODO(), resource)
	if err != nil {
		klog.Errorf("unable to delete resource %s", resource.GetName())
		return err
	}
	klog.Infof("deleted k8s network policy %s", resource.GetName())
	return nil
}
