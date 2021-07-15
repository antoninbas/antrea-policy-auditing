package test

var (
    Dir = ""
	SelectorA = metav1.LabelSelector{MatchLabels: map[string]string{"foo1": "bar1"}}
	SelectorB = metav1.LabelSelector{MatchLabels: map[string]string{"foo2": "bar2"}}
	SelectorC = metav1.LabelSelector{MatchLabels: map[string]string{"foo3": "bar3"}}
	P10 = float64(10)
	Int80 = intstr.FromInt(80)
	Int81 = intstr.FromInt(81)
	AllowAction = crdv1alpha1.RuleActionAllow
	Np1 = test_resource{
		inputResource: &networkingv1.NetworkPolicy{
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
	Np2 = test_resource{
		inputResource: &networkingv1.NetworkPolicy{
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
	Np3 = test_resource{
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
	Acnp1 = test_resource{
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
	Tier1 = test_resource{
		inputResource: &crdv1alpha1.Tier{
			ObjectMeta: metav1.ObjectMeta{
				Name: "TierA",
			},
			Spec: crdv1alpha1.TierSpec{
				Priority: 10,
				Description: "This is a test tier",
			},
		},
		expPath: "/antrea-tiers/TierA.yaml",
		expYaml:
`apiVersion: crd.antrea.io/v1alpha1
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
