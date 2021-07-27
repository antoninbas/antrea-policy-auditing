package gitops

import (
	"context"
	"os"
	"path/filepath"

	v1alpha1 "antrea.io/antrea/pkg/apis/crd/v1alpha1"
	networking "k8s.io/api/networking/v1"
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
	Client client.Client
}

func NewKubernetes() (*K8sClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := filepath.Join(
			os.Getenv("KUBECONFIG"),
		)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			klog.ErrorS(err, "unable to build config from flags, check that your KUBECONFIG file is correct!")
			return nil, err
		}
	}
	scheme := runtime.NewScheme()
	registerTypes(scheme)
	client, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		klog.ErrorS(err, "unable to instantiate new generic client")
		return nil, err
	}
	return &K8sClient{Client: client}, nil
}

func registerTypes(scheme *runtime.Scheme) {
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group: "networking.k8s.io", 
		Version: "v1", 
		Kind: "NetworkPolicyList"}, 
		&networking.NetworkPolicyList{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group: "networking.k8s.io", 
		Version: "v1", 
		Kind: "ListOptions"}, 
		&metav1.ListOptions{})	
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group: "crd.antrea.io", 
		Version: "v1alpha1", 
		Kind: "NetworkPolicyList"}, 
		&v1alpha1.NetworkPolicyList{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group: "crd.antrea.io", 
		Version: "v1alpha1", 
		Kind: "ClusterNetworkPolicyList"}, 
		&v1alpha1.ClusterNetworkPolicyList{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group: "crd.antrea.io", 
		Version: "v1alpha1", 
		Kind: "TierList"}, 
		&v1alpha1.TierList{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group: "crd.antrea.io", 
		Version: "v1alpha1", 
		Kind: "ListOptions"},
		&metav1.ListOptions{})
}

func (k *K8sClient) ListResource(resourceList *unstructured.UnstructuredList) (*unstructured.UnstructuredList, error) {
	err := k.Client.List(context.TODO(), resourceList)
	if err != nil {
		klog.ErrorS(err, "unable to list resource")
		return nil, err
	}
	return resourceList, nil
}

func (k *K8sClient) CreateOrUpdateResource(resource *unstructured.Unstructured) error {
	if err := k.Client.Create(context.TODO(), resource); err == nil {
		klog.V(2).Infof("created resource %s", resource.GetName())
		return nil
	}
	klog.V(2).Infof("unable to create resource, trying update instead")
	oldResource := &unstructured.Unstructured{}
	_ = k.Client.Get(context.TODO(), client.ObjectKey{
		Namespace: resource.GetNamespace(),
		Name: resource.GetName(),
	}, oldResource)
	resource.SetResourceVersion(oldResource.GetResourceVersion())
	if err := k.Client.Update(context.TODO(), resource); err != nil {
		klog.Errorf("unable to update k8s resource %s", resource.GetName())
		return err
	}
	klog.V(2).Infof("updated resource %s", resource.GetName())
	return nil
}

func (k *K8sClient) DeleteResource(resource *unstructured.Unstructured) error {
	err := k.Client.Delete(context.TODO(), resource)
	if err != nil {
		klog.Errorf("unable to delete resource %s", resource.GetName())
		return err
	}
	klog.V(2).Infof("deleted k8s network policy %s", resource.GetName())
	return nil
}
