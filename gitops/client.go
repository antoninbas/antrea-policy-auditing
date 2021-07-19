package gitops

import (
	"context"
	"os"
	"path/filepath"

	v1alpha1 "antrea.io/antrea/pkg/apis/crd/v1alpha1"
	crdclientset "antrea.io/antrea/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

type KubeClients struct {
	ClientSet kubernetes.Interface
	CrdClient crdclientset.Interface
}

func NewKubernetes() (*KubeClients, error) {
	clientSet, crdClientSet, err := Client()
	if err != nil {
		return nil, errors.WithMessagef(err, "unable to instantiate clientsets")
	}
	return &KubeClients{
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

func (k *KubeClients) GetK8sPolicies() (*networking.NetworkPolicyList, error) {
	l, err := k.ClientSet.NetworkingV1().NetworkPolicies("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.ErrorS(err, "unable to list k8s network policies")
		return nil, err
	}
	return l, nil
}

func (k *KubeClients) CreateOrUpdateK8sPolicy(policy *networking.NetworkPolicy) error {
	_, err := k.ClientSet.NetworkingV1().NetworkPolicies(policy.Namespace).Create(context.TODO(), policy, metav1.CreateOptions{})
	if err == nil {
		klog.V(2).Infof("created k8s network policy %s in namespace %s", policy.Name, policy.Namespace)
		return nil
	}
	klog.V(2).Infof("unable to create k8s network policy %s in namespace %s, trying update instead", policy.Name, policy.Namespace)
	_, err = k.ClientSet.NetworkingV1().NetworkPolicies(policy.Namespace).Update(context.TODO(), policy, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("unable to create/update k8s network policy %s in namespace %s", policy.Name, policy.Namespace)
		return err
	}
	klog.V(2).Infof("updated k8s network policy %s in namespace %s", policy.Name, policy.Namespace)
	return nil
}

func (k *KubeClients) DeleteK8sPolicy(policy *networking.NetworkPolicy) error {
	err := k.ClientSet.NetworkingV1().NetworkPolicies(policy.Namespace).Delete(context.TODO(), policy.Name, metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("unable to delete k8s network policy %s in namespace %s", policy.Name, policy.Namespace)
		return err
	}
	klog.V(2).Infof("deleted k8s network policy %s in namespace %s", policy.Name, policy.Namespace)
	return nil
}

func (k *KubeClients) GetAntreaPolicies() (*v1alpha1.NetworkPolicyList, error) {
	l, err := k.CrdClient.CrdV1alpha1().NetworkPolicies("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.ErrorS(err, "unable to list antrea network policies")
		return nil, err
	}
	return l, nil
}

func (k *KubeClients) CreateOrUpdateAntreaPolicy(policy *v1alpha1.NetworkPolicy) error {
	_, err := k.CrdClient.CrdV1alpha1().NetworkPolicies(policy.Namespace).Create(context.TODO(), policy, metav1.CreateOptions{})
	if err == nil {
		klog.V(2).Infof("created antrea network policy %s in namespace %s", policy.Name, policy.Namespace)
		return nil
	}
	klog.V(2).Infof("unable to create antrea network policy %s in namespace %s, trying update instead", policy.Name, policy.Namespace)
	_, err = k.CrdClient.CrdV1alpha1().NetworkPolicies(policy.Namespace).Update(context.TODO(), policy, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("unable to create/update antrea network policy %s in namespace %s", policy.Name, policy.Namespace)
		return err
	}
	klog.V(2).Infof("updated antrea network policy %s in namespace %s", policy.Name, policy.Namespace)
	return nil
}

func (k *KubeClients) DeleteAntreaPolicy(policy *v1alpha1.NetworkPolicy) error {
	err := k.CrdClient.CrdV1alpha1().NetworkPolicies(policy.Namespace).Delete(context.TODO(), policy.Name, metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("unable to delete antrea network policy %s in namespace %s", policy.Name, policy.Namespace)
		return err
	}
	klog.V(2).Infof("deleted antrea network policy %s in namespace %s", policy.Name, policy.Namespace)
	return nil
}

func (k *KubeClients) GetAntreaClusterPolicies() (*v1alpha1.ClusterNetworkPolicyList, error) {
	l, err := k.CrdClient.CrdV1alpha1().ClusterNetworkPolicies().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.ErrorS(err, "unable to list antrea cluster network policies")
		return nil, err
	}
	return l, nil
}

func (k *KubeClients) CreateOrUpdateAntreaClusterPolicy(policy *v1alpha1.ClusterNetworkPolicy) error {
	_, err := k.CrdClient.CrdV1alpha1().ClusterNetworkPolicies().Create(context.TODO(), policy, metav1.CreateOptions{})
	if err == nil {
		klog.V(2).Infof("created antrea cluster network policy %s", policy.Name)
		return nil
	}
	klog.V(2).Infof("unable to create antrea cluster network policy %s, trying update instead", policy.Name)
	_, err = k.CrdClient.CrdV1alpha1().ClusterNetworkPolicies().Update(context.TODO(), policy, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("unable to create/update antrea cluster network policy %s", policy.Name)
		return err
	}
	klog.V(2).Infof("updated antrea cluster network policy %s", policy.Name)
	return nil
}

func (k *KubeClients) DeleteAntreaClusterPolicy(policy *v1alpha1.ClusterNetworkPolicy) error {
	err := k.CrdClient.CrdV1alpha1().ClusterNetworkPolicies().Delete(context.TODO(), policy.Name, metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("unable to delete antrea cluster network policy %s", policy.Name)
		return err
	}
	klog.V(2).Infof("deleted antrea cluster network policy %s", policy.Name)
	return nil
}

func (k *KubeClients) GetAntreaTiers() (*v1alpha1.TierList, error) {
	l, err := k.CrdClient.CrdV1alpha1().Tiers().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.ErrorS(err, "unable to list antrea tiers")
		return nil, err
	}
	return l, nil
}

func (k *KubeClients) CreateOrUpdateAntreaTier(tier *v1alpha1.Tier) error {
	_, err := k.CrdClient.CrdV1alpha1().Tiers().Create(context.TODO(), tier, metav1.CreateOptions{})
	if err == nil {
		klog.V(2).Infof("created antrea tier %s", tier.Name)
		return nil
	}
	klog.V(2).Infof("unable to create antrea tier %s, trying update instead", tier.Name)
	_, err = k.CrdClient.CrdV1alpha1().Tiers().Update(context.TODO(), tier, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("unable to create/update antrea tier %s", tier.Name)
		return err
	}
	klog.V(2).Infof("updated antrea tier %s", tier.Name)
	return nil
}

func (k *KubeClients) DeleteAntreaTier(tier *v1alpha1.Tier) error {
	err := k.CrdClient.CrdV1alpha1().Tiers().Delete(context.TODO(), tier.Name, metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("unable to delete antrea tier %s", tier.Name)
		return err
	}
	klog.V(2).Infof("deleted antrea tier %s", tier.Name)
	return nil
}
