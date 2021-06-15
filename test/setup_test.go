package test

import (
    "testing"
	"os"

	v1 "k8s.io/api/core/v1"
	. "antrea-audit/git-manager/init"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/runtime"
	fake "k8s.io/client-go/kubernetes/fake"
	fakeversioned "antrea.io/antrea/pkg/client/clientset/versioned/fake"
	crdv1alpha1 "antrea.io/antrea/pkg/apis/crd/v1alpha1"
)

var (
	selectorA = metav1.LabelSelector{MatchLabels: map[string]string{"foo1": "bar1"}}
	selectorB = metav1.LabelSelector{MatchLabels: map[string]string{"foo2": "bar2"}}
	selectorC = metav1.LabelSelector{MatchLabels: map[string]string{"foo3": "bar3"}}
	p10 = float64(10)
	int80 = intstr.FromInt(80)
	int81 = intstr.FromInt(81)
	allowAction = crdv1alpha1.RuleActionAllow
	np1 = &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "npA", UID: "uidA"},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress:     []networkingv1.NetworkPolicyIngressRule{{}},
		},
	}
	np2 = &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "npB", UID: "uidB"},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress:      []networkingv1.NetworkPolicyEgressRule{{}},
		},
	}
	np3 = &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "nsB", Name: "npC", UID: "uidC"},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: selectorA,
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Port: &int80,
						},
					},
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector:       &selectorB,
							NamespaceSelector: &selectorC,
						},
					},
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Port: &int81,
						},
					},
					To: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector:       &selectorB,
							NamespaceSelector: &selectorC,
						},
					},
				},
			},
		},
	}
	anp1 = &crdv1alpha1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "npA", UID: "uidA"},
		Spec: crdv1alpha1.NetworkPolicySpec{
			AppliedTo: []crdv1alpha1.NetworkPolicyPeer{
				{PodSelector: &selectorA},
			},
			Priority: p10,
			Ingress: []crdv1alpha1.Rule{
				{
					Ports: []crdv1alpha1.NetworkPolicyPort{
						{
							Port: &int80,
						},
					},
					From: []crdv1alpha1.NetworkPolicyPeer{
						{
							PodSelector:       &selectorB,
							NamespaceSelector: &selectorC,
						},
					},
					Action: &allowAction,
				},
			},
			Egress: []crdv1alpha1.Rule{
				{
					Ports: []crdv1alpha1.NetworkPolicyPort{
						{
							Port: &int81,
						},
					},
					To: []crdv1alpha1.NetworkPolicyPeer{
						{
							PodSelector:       &selectorB,
							NamespaceSelector: &selectorC,
						},
					},
					Action: &allowAction,
				},
			},
		},
	}
	acnp1 = &crdv1alpha1.ClusterNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "cnpA", UID: "uidA"},
		Spec: crdv1alpha1.ClusterNetworkPolicySpec{
			AppliedTo: []crdv1alpha1.NetworkPolicyPeer{
				{PodSelector: &selectorA},
			},
			Priority: p10,
			Ingress: []crdv1alpha1.Rule{
				{
					Ports: []crdv1alpha1.NetworkPolicyPort{
						{
							Port: &int80,
						},
					},
					From: []crdv1alpha1.NetworkPolicyPeer{
						{
							PodSelector:       &selectorB,
							NamespaceSelector: &selectorC,
						},
					},
					Action: &allowAction,
				},
			},
			Egress: []crdv1alpha1.Rule{
				{
					Ports: []crdv1alpha1.NetworkPolicyPort{
						{
							Port: &int81,
						},
					},
					To: []crdv1alpha1.NetworkPolicyPeer{
						{
							PodSelector:       &selectorB,
							NamespaceSelector: &selectorC,
						},
					},
					Action: &allowAction,
				},
			},
		},
	}
)

func TestSetupRepo(t *testing.T) {
	tests := []struct {
		name string
		inputK8sPolicies []runtime.Object
		inputCRDPolicies []runtime.Object
		expectedPaths []string
	}{
		{
			name: "empty-test",
			inputK8sPolicies: []runtime.Object{},
			inputCRDPolicies: []runtime.Object{},
			expectedPaths: []string{},
		},
		{
			name: "basic-test",
			inputK8sPolicies: []runtime.Object{np1, np2, np3},
			inputCRDPolicies: []runtime.Object{anp1, acnp1},
			expectedPaths: []string{"/k8s-policy/nsA/npA.yaml", "/k8s-policy/nsA/npB.yaml", "/k8s-policy/nsB/npC.yaml", "/antrea-policy/nsA/npA.yaml", "/antrea-cluster-policy/cnpA.yaml"},
		},
		{
			name: "empty-K8s-test",
			inputK8sPolicies: []runtime.Object{},
			inputCRDPolicies: []runtime.Object{anp1, acnp1},
			expectedPaths: []string{"/antrea-policy/nsA/npA.yaml", "/antrea-cluster-policy/cnpA.yaml"},
		},
		{
			name: "empty-CRDs-test",
			inputK8sPolicies: []runtime.Object{np1, np2},
			inputCRDPolicies: []runtime.Object{},
			expectedPaths: []string{"/k8s-policy/nsA/npA.yaml", "/k8s-policy/nsA/npB.yaml"},
		},
	}
	for _, test := range tests {
		fakeK8sClient := newK8sClientSet(test.inputK8sPolicies...)
		fakeCRDClient := newCRDClientSet(test.inputCRDPolicies...)
		runTest(t, fakeK8sClient, fakeCRDClient, test.expectedPaths)
	}
}

func runTest(t *testing.T, K8sClient *fake.Clientset, CRDClient *fakeversioned.Clientset, expPaths []string) {
	k8s := &Kubernetes{
		PodCache:  map[string][]v1.Pod{},
		ClientSet: K8sClient,
		CrdClient: CRDClient,
	}

	if err := SetupRepo(k8s); err != nil {
		t.Errorf("Error (TestSetupRepo): unable to set up repo")
	}

	for _, path := range expPaths {
		cwd, err := os.Getwd()
		cwd += "/network-policy-repository"
        if err != nil {
            t.Errorf("Error (TestSetupRepo): could not retrieve the current working directory")
        }
		if _, err := os.Stat(cwd + path); os.IsNotExist(err) {
			t.Errorf("Error (TestSetupRepo): file was not found in the correct repo subdirectory")
		  }
	} 

	//TODO: verify that yamls contain correct contents

	os.RemoveAll("./network-policy-repository")
}

func newK8sClientSet(objects ...runtime.Object) *fake.Clientset {
	client := fake.NewSimpleClientset(objects...)
	return client
}

func newCRDClientSet(objects ...runtime.Object) *fakeversioned.Clientset {
	client := fakeversioned.NewSimpleClientset(objects ...)
	return client
}
