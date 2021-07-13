package test

import (
	"testing"

	"antrea-audit/gitops"

	crdv1alpha1 "antrea.io/antrea/pkg/apis/crd/v1alpha1"
	fakeversioned "antrea.io/antrea/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	fake "k8s.io/client-go/kubernetes/fake"
)

type test_resource struct {
	inputResource runtime.Object
	expPath       string
	expYaml       string
}

var (
	dir         = ""
	selectorA   = metav1.LabelSelector{MatchLabels: map[string]string{"foo1": "bar1"}}
	selectorB   = metav1.LabelSelector{MatchLabels: map[string]string{"foo2": "bar2"}}
	selectorC   = metav1.LabelSelector{MatchLabels: map[string]string{"foo3": "bar3"}}
	p10         = float64(10)
	int80       = intstr.FromInt(80)
	int81       = intstr.FromInt(81)
	allowAction = crdv1alpha1.RuleActionAllow
	Np1         = test_resource{
		inputResource: &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "npA", UID: "uidA"},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
				Ingress:     []networkingv1.NetworkPolicyIngressRule{{}},
			},
		},
		expPath: "/k8s-policies/nsA/npA.yaml",
		expYaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  creationTimestamp: null
  name: npA
  namespace: nsA
  uid: uidA
spec:
  ingress:
  - {}
  podSelector: {}
  policyTypes:
  - Ingress
`,
	}
	np2 = test_resource{
		inputResource: &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "npB", UID: "uidB"},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
				Egress:      []networkingv1.NetworkPolicyEgressRule{{}},
			},
		},
		expPath: "/k8s-policies/nsA/npB.yaml",
		expYaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  creationTimestamp: null
  name: npB
  namespace: nsA
  uid: uidB
spec:
  egress:
  - {}
  podSelector: {}
  policyTypes:
  - Egress
`,
	}
	np3 = test_resource{
		inputResource: &networkingv1.NetworkPolicy{
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
		},
		expPath: "/k8s-policies/nsB/npC.yaml",
		expYaml: `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  creationTimestamp: null
  name: npC
  namespace: nsB
  uid: uidC
spec:
  egress:
  - ports:
    - port: 81
    to:
    - namespaceSelector:
        matchLabels:
          foo3: bar3
      podSelector:
        matchLabels:
          foo2: bar2
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          foo3: bar3
      podSelector:
        matchLabels:
          foo2: bar2
    ports:
    - port: 80
  podSelector:
    matchLabels:
      foo1: bar1
`,
	}
	Anp1 = test_resource{
		inputResource: &crdv1alpha1.NetworkPolicy{
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
		},
		expPath: "/antrea-policies/nsA/npA.yaml",
		expYaml: `apiVersion: crd.antrea.io/v1alpha1
kind: NetworkPolicy
metadata:
  creationTimestamp: null
  name: npA
  namespace: nsA
  uid: uidA
spec:
  appliedTo:
  - podSelector:
      matchLabels:
        foo1: bar1
  egress:
  - action: Allow
    enableLogging: false
    from: null
    name: ""
    ports:
    - port: 81
    to:
    - namespaceSelector:
        matchLabels:
          foo3: bar3
      podSelector:
        matchLabels:
          foo2: bar2
  ingress:
  - action: Allow
    enableLogging: false
    from:
    - namespaceSelector:
        matchLabels:
          foo3: bar3
      podSelector:
        matchLabels:
          foo2: bar2
    name: ""
    ports:
    - port: 80
    to: null
  priority: 10
status:
  currentNodesRealized: 0
  desiredNodesRealized: 0
  observedGeneration: 0
  phase: ""
`,
	}
	acnp1 = test_resource{
		inputResource: &crdv1alpha1.ClusterNetworkPolicy{
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
		},
		expPath: "/antrea-cluster-policies/cnpA.yaml",
		expYaml: `apiVersion: crd.antrea.io/v1alpha1
kind: ClusterNetworkPolicy
metadata:
  creationTimestamp: null
  name: cnpA
  uid: uidA
spec:
  appliedTo:
  - podSelector:
      matchLabels:
        foo1: bar1
  egress:
  - action: Allow
    enableLogging: false
    from: null
    name: ""
    ports:
    - port: 81
    to:
    - namespaceSelector:
        matchLabels:
          foo3: bar3
      podSelector:
        matchLabels:
          foo2: bar2
  ingress:
  - action: Allow
    enableLogging: false
    from:
    - namespaceSelector:
        matchLabels:
          foo3: bar3
      podSelector:
        matchLabels:
          foo2: bar2
    name: ""
    ports:
    - port: 80
    to: null
  priority: 10
status:
  currentNodesRealized: 0
  desiredNodesRealized: 0
  observedGeneration: 0
  phase: ""
`,
	}
	tier1 = test_resource{
		inputResource: &crdv1alpha1.Tier{
			ObjectMeta: metav1.ObjectMeta{
				Name: "TierA",
			},
			Spec: crdv1alpha1.TierSpec{
				Priority:    10,
				Description: "This is a test tier",
			},
		},
		expPath: "/antrea-tiers/TierA.yaml",
		expYaml: `apiVersion: crd.antrea.io/v1alpha1
kind: Tier
metadata:
  creationTimestamp: null
  name: TierA
spec:
  description: This is a test tier
  priority: 10
`,
	}
)

func TestSetupRepo(t *testing.T) {
	tests := []struct {
		name              string
		inputK8sResources []test_resource
		inputCRDResources []test_resource
	}{
		{
			name:              "empty-test",
			inputK8sResources: []test_resource{},
			inputCRDResources: []test_resource{},
		},
		{
			name:              "basic-test",
			inputK8sResources: []test_resource{Np1, np2, np3},
			inputCRDResources: []test_resource{Anp1, acnp1},
		},
		{
			name:              "empty-K8s-test",
			inputK8sResources: []test_resource{},
			inputCRDResources: []test_resource{Anp1, acnp1},
		},
		{
			name:              "empty-CRDs-test",
			inputK8sResources: []test_resource{Np1, np2},
			inputCRDResources: []test_resource{},
		},
		{
			name:              "tiers-test",
			inputK8sResources: []test_resource{Np1, np2},
			inputCRDResources: []test_resource{Anp1, tier1},
		},
	}
	for _, test := range tests {
		var expectedPaths = []string{}
		var expectedYamls = []string{}
		var k8sResources = []runtime.Object{}
		for _, resource := range test.inputK8sResources {
			k8sResources = append(k8sResources, resource.inputResource)
			expectedPaths = append(expectedPaths, resource.expPath)
			expectedYamls = append(expectedYamls, resource.expYaml)
		}
		var crdResources = []runtime.Object{}
		for _, resource := range test.inputCRDResources {
			crdResources = append(crdResources, resource.inputResource)
			expectedPaths = append(expectedPaths, resource.expPath)
			expectedYamls = append(expectedYamls, resource.expYaml)
		}
		fakeK8sClient := NewK8sClientSet(k8sResources...)
		fakeCRDClient := NewCRDClientSet(crdResources...)
		k8s := &gitops.Kubernetes{
			PodCache:  map[string][]v1.Pod{},
			ClientSet: fakeK8sClient,
			CrdClient: fakeCRDClient,
		}
		runSetupTest(t, k8s, expectedPaths, expectedYamls)
	}
}

func runSetupTest(t *testing.T, k8s *gitops.Kubernetes, expPaths []string, expYamls []string) {
	cr, err := gitops.SetupRepo(k8s, "mem", dir)
	if err != nil {
		t.Errorf("Error (TestSetupRepo): unable to set up repo")
	}
	for i, path := range expPaths {
		file, err := cr.Fs.Open(path)
		if err != nil {
			t.Errorf("Error (TestSetupRepo): unable to open file")
		}
		fstat, _ := cr.Fs.Stat(path)
		var buffer = make([]byte, fstat.Size())
		file.Read(buffer)
		assert.Equal(t, string(buffer), expYamls[i], "Error (TestSetupRepo): file does not match expected YAML")
	}
}

func TestRepoDuplicate(t *testing.T) {
	fakeK8sClient := NewK8sClientSet(Np1.inputResource)
	fakeCRDClient := NewCRDClientSet(Anp1.inputResource)
	k8s := &gitops.Kubernetes{
		PodCache:  map[string][]v1.Pod{},
		ClientSet: fakeK8sClient,
		CrdClient: fakeCRDClient,
	}
	_, err := gitops.SetupRepo(k8s, "mem", dir)
	if err != nil {
		t.Errorf("Error (TestRepoDuplicate): unable to set up repo for the first time")
	}
	_, err = gitops.SetupRepo(k8s, "mem", dir)
	if err != nil {
		t.Errorf("Error (TestRepoDuplicate): unable to set up repo for the second time")
	}
}

func NewK8sClientSet(objects ...runtime.Object) *fake.Clientset {
	client := fake.NewSimpleClientset(objects...)
	return client
}

func NewCRDClientSet(objects ...runtime.Object) *fakeversioned.Clientset {
	client := fakeversioned.NewSimpleClientset(objects...)
	return client
}
