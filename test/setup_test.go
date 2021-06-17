package test

import (
	"fmt"
    "testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	memfs "github.com/go-git/go-billy/v5/memfs"
	memory "github.com/go-git/go-git/v5/storage/memory"
	v1 "k8s.io/api/core/v1"
	. "antrea-audit/git-manager/init"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/go-git/go-git/v5"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/runtime"
	fake "k8s.io/client-go/kubernetes/fake"
	fakeversioned "antrea.io/antrea/pkg/client/clientset/versioned/fake"
	crdv1alpha1 "antrea.io/antrea/pkg/apis/crd/v1alpha1"
)

type test_policy struct {
	inputPolicy runtime.Object
	expPath string
	expYaml string
}

var (
	selectorA = metav1.LabelSelector{MatchLabels: map[string]string{"foo1": "bar1"}}
	selectorB = metav1.LabelSelector{MatchLabels: map[string]string{"foo2": "bar2"}}
	selectorC = metav1.LabelSelector{MatchLabels: map[string]string{"foo3": "bar3"}}
	p10 = float64(10)
	int80 = intstr.FromInt(80)
	int81 = intstr.FromInt(81)
	allowAction = crdv1alpha1.RuleActionAllow
	np1 = test_policy{
		inputPolicy: &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "npA", UID: "uidA"},
				Spec: networkingv1.NetworkPolicySpec{
					PodSelector: metav1.LabelSelector{},
					PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
					Ingress:     []networkingv1.NetworkPolicyIngressRule{{}},
				},
		},
		expPath: "/k8s-policies/nsA/npA.yaml",
		expYaml: 
`apiVersion: networking.k8s.io/v1
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
	np2 = test_policy{
		inputPolicy: &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "npB", UID: "uidB"},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
				Egress:      []networkingv1.NetworkPolicyEgressRule{{}},
			},
		},
		expPath: "/k8s-policies/nsA/npB.yaml",
		expYaml: 
`apiVersion: networking.k8s.io/v1
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
	np3 = test_policy{
		inputPolicy: &networkingv1.NetworkPolicy{
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
		expYaml: 
`apiVersion: networking.k8s.io/v1
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
	anp1 = test_policy{
		inputPolicy: &crdv1alpha1.NetworkPolicy{
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
		expYaml: 
`apiVersion: crd.antrea.io/v1alpha1
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
	acnp1 = test_policy{
		inputPolicy: &crdv1alpha1.ClusterNetworkPolicy{
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
		expYaml: 
`apiVersion: crd.antrea.io/v1alpha1
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
)

func TestSetupRepo(t *testing.T) {
	tests := []struct {
		name string
		inputK8sPolicies []test_policy
		inputCRDPolicies []test_policy
	}{
		{
			name: "empty-test",
			inputK8sPolicies: []test_policy{},
			inputCRDPolicies: []test_policy{},
		},
		{
			name: "basic-test",
			inputK8sPolicies: []test_policy{np1, np2, np3},
			inputCRDPolicies: []test_policy{anp1, acnp1},
		},
		{
			name: "empty-K8s-test",
			inputK8sPolicies: []test_policy{},
			inputCRDPolicies: []test_policy{anp1, acnp1},
		},
		{
			name: "empty-CRDs-test",
			inputK8sPolicies: []test_policy{np1, np2},
			inputCRDPolicies: []test_policy{},
		},
	}
	for _, test := range tests {
		var expectedPaths = []string{}
		var expectedYamls = []string{}
		var k8sPolicies = []runtime.Object{}
		for _, policy := range test.inputK8sPolicies {
			k8sPolicies = append(k8sPolicies, policy.inputPolicy)
			expectedPaths = append(expectedPaths, policy.expPath)
			expectedYamls = append(expectedYamls, policy.expYaml)
		}
		var crdPolicies = []runtime.Object{}
		for _, policy := range test.inputCRDPolicies {
			crdPolicies = append(crdPolicies, policy.inputPolicy)
			expectedPaths = append(expectedPaths, policy.expPath)
			expectedYamls = append(expectedYamls, policy.expYaml)
		}
		fakeK8sClient := newK8sClientSet(k8sPolicies...)
		fakeCRDClient := newCRDClientSet(crdPolicies...)
		k8s := &Kubernetes{
			PodCache:  map[string][]v1.Pod{},
			ClientSet: fakeK8sClient,
			CrdClient: fakeCRDClient,
		}
		runTest(t, k8s, expectedPaths, expectedYamls)
	}
}

func runTest(t *testing.T, k8s *Kubernetes, expPaths []string, expYamls []string) {
	storer := memory.NewStorage()
	fs := memfs.New()
	if err := SetupRepoInMem(k8s, storer, fs); err != nil {
		t.Errorf("Error (TestSetupRepo): unable to set up repo")
	}
	for i, path := range expPaths {
		fmt.Println(path)
		file, err := fs.Open(path)
		if err != nil {
			t.Errorf("Error (TestSetupRepo): unable to open file")
		}
		fstat, _ := fs.Stat(path)
		var buffer = make([]byte, fstat.Size())
		file.Read(buffer)
		assert.Equal(t, string(buffer), expYamls[i], "Error (TestSetupRepo): file does not match expected YAML")
	}
}

func TestRepoDuplicate(t *testing.T) {
	fakeK8sClient := newK8sClientSet(np1.inputPolicy)
	fakeCRDClient := newCRDClientSet(anp1.inputPolicy)
	k8s := &Kubernetes{
		PodCache:  map[string][]v1.Pod{},
		ClientSet: fakeK8sClient,
		CrdClient: fakeCRDClient,
	}
	storer := memory.NewStorage()
	fs := memfs.New()
	if err := SetupRepoInMem(k8s, storer, fs); err != nil {
		t.Errorf("Error (TestRepoDuplicate): unable to set up repo for the first time")
	}
	if err := SetupRepoInMem(k8s, storer, fs); errors.Cause(err) != git.ErrRepositoryAlreadyExists {
		t.Errorf("Error (TestRepoDuplicate): should have detected that repo already exists")
	}
}

func newK8sClientSet(objects ...runtime.Object) *fake.Clientset {
	client := fake.NewSimpleClientset(objects...)
	return client
}

func newCRDClientSet(objects ...runtime.Object) *fakeversioned.Clientset {
	client := fakeversioned.NewSimpleClientset(objects ...)
	return client
}
