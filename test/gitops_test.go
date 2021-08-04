package test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"antrea-audit/gitops"

	crdv1alpha1 "antrea.io/antrea/pkg/apis/crd/v1alpha1"
	billy "github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	memory "github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	np1 = &networkingv1.NetworkPolicy{
		TypeMeta:   metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "networking.k8s.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "npA", UID: "uidA"},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress:     []networkingv1.NetworkPolicyIngressRule{{}},
		},
	}
	np2 = &networkingv1.NetworkPolicy{
		TypeMeta:   metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "networking.k8s.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "npB", UID: "uidB"},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress:      []networkingv1.NetworkPolicyEgressRule{{}},
		},
	}
	anp1 = &crdv1alpha1.NetworkPolicy{
		TypeMeta:   metav1.TypeMeta{Kind: "NetworkPolicy", APIVersion: "crd.antrea.io/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "nsA", Name: "anpA", UID: "uidC"},
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
)

func TestHandleEventList(t *testing.T) {
	fakeClient := NewClient(np1.DeepCopy(), anp1.DeepCopy())
	k8s := &gitops.K8sClient{
		Client: fakeClient,
	}

	jsonstring, err := ioutil.ReadFile("./files/correct-audit-log.txt")
	assert.NoError(t, err, "unable to read mock audit log")

	cr, err := gitops.SetupRepo(k8s, gitops.StorageModeInMemory, dir)
	assert.NoError(t, err, "could not set up repo")

	err = cr.HandleEventList(jsonstring)
	assert.NoError(t, err, "could not handle correct audit event list")

	cr.RollbackMode = true
	err = cr.HandleEventList(jsonstring)
	cr.RollbackMode = false
	assert.EqualError(t, err, "audit skipped - rollback in progress")

	for i := 1; i < 4; i++ {
		filename := fmt.Sprintf("%s%d%s", "files/incorrect-audit-log-", i, ".txt")
		jsonstring, err := ioutil.ReadFile(filename)
		assert.NoError(t, err, "unable to read audit log")
		err = cr.HandleEventList(jsonstring)
		assert.Error(t, err, fmt.Sprintf("should have returned error on bad audit log: %d", i))
	}
}

func TestTagging(t *testing.T) {
	fakeClient := NewClient()
	k8s := &gitops.K8sClient{
		Client: fakeClient,
	}
	cr, err := gitops.SetupRepo(k8s, gitops.StorageModeInMemory, dir)
	assert.NoError(t, err, "unable to set up repo")
	h, err := cr.Repo.Head()
	assert.NoError(t, err, "unable to get repo head ref")

	testSig := &object.Signature{
		Name:  "test",
		Email: "test@antrea.audit.io",
		When:  time.Now(),
	}
	// Attempt to add tag to nonexistent commit
	err = cr.TagCommit("bad-hash", "test-tag", testSig)
	assert.Error(t, err, "should have returned error on bad commit hash")

	// Create new tags successfully
	err = cr.TagCommit(h.Hash().String(), "test-tag", testSig)
	assert.NoError(t, err, "unable to create 1st new tag")
	err = cr.TagCommit(h.Hash().String(), "test-tag-2", testSig)
	assert.NoError(t, err, "unable to create 2nd new tag")

	_, err = cr.Repo.Tag("test-tag")
	assert.NoError(t, err, "could not retrieve 1st created tag")
	_, err = cr.Repo.Tag("test-tag-2")
	assert.NoError(t, err, "could not retrieve 2nd created tag")

	// Attempt to add tag with the same name
	err = cr.TagCommit(h.Hash().String(), "test-tag", testSig)
	assert.EqualError(t, err, "unable to create tag: tag already exists")

	tags, _ := cr.Repo.TagObjects()
	tagCount := 0
	err = tags.ForEach(func(tag *object.Tag) error {
		tagCount += 1
		return nil
	})
	assert.NoError(t, err, "could not iterate through repo tags")
	assert.Equal(t, 2, tagCount, "unexpected number of tags, should have 2 tags")
}

func TestRollback(t *testing.T) {
	fakeClient := NewClient(np1.DeepCopy(), anp1.DeepCopy())
	k8s := &gitops.K8sClient{
		Client: fakeClient,
	}
	cr, err := gitops.SetupRepo(k8s, gitops.StorageModeInMemory, dir)
	assert.NoError(t, err, "unable to set up repo")
	h, err := cr.Repo.Head()
	assert.NoError(t, err, "unable to get repo head ref")
	testSig := &object.Signature{
		Name:  "test",
		Email: "test@antrea.audit.io",
		When:  time.Now(),
	}
	err = cr.TagCommit(h.Hash().String(), "test-tag", testSig)
	assert.NoError(t, err, "unable to create new tag")

	// Create, update, and delete a resource
	r := unstructured.Unstructured{}
	r.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.k8s.io",
		Version: "v1",
		Kind:    "NetworkPolicy",
	})
	j, err := json.Marshal(np2)
	assert.NoError(t, err, "unable to convert to json")
	err = json.Unmarshal(j, &r)
	assert.NoError(t, err, "unable to unmarshal into unstructured object")
	err = k8s.CreateOrUpdateResource(&r)
	assert.NoError(t, err, "unable to create new resource")

	updatedNP := np1
	updatedNP.ObjectMeta.SetClusterName("new-cluster-name")
	r = unstructured.Unstructured{}
	r.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.k8s.io",
		Version: "v1",
		Kind:    "NetworkPolicy",
	})
	j, err = json.Marshal(updatedNP)
	assert.NoError(t, err, "unable to convert to json")
	err = json.Unmarshal(j, &r)
	assert.NoError(t, err, "unable to unmarshal into structured object")
	err = k8s.CreateOrUpdateResource(&r)
	assert.NoError(t, err, "unable to update new resource")

	r = unstructured.Unstructured{}
	r.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "NetworkPolicy",
	})
	j, err = json.Marshal(anp1)
	assert.NoError(t, err, "unable to convert to json")
	err = json.Unmarshal(j, &r)
	assert.NoError(t, err, "unable to unmarshal into structured object")
	err = k8s.DeleteResource(&r)
	assert.NoError(t, err, "unable to delete resource")

	jsonStr, err := ioutil.ReadFile("./files/rollback-log.txt")
	assert.NoError(t, err, "could not read rollback-log file")
	err = cr.HandleEventList(jsonStr)
	assert.NoError(t, err, "could not process audit events from file")

	// Attempt rollback
	commit, err := cr.TagToCommit("test-tag")
	assert.NoError(t, err, "could not retrieve commit from tag")
	err = cr.RollbackRepo(commit)
	assert.NoError(t, err, "rollback failed")

	// Check latest commit
	newH, err := cr.Repo.Head()
	assert.NoError(t, err, "unable to get repo head ref")
	rollbackCommit, err := cr.Repo.CommitObject(newH.Hash())
	assert.NoError(t, err, "unable to get rollback commit object")
	assert.Equal(t, "Rollback to commit "+h.Hash().String(), rollbackCommit.Message,
		"rollback commit not found, head commit message mismatch")

	// Check cluster state
	res := &unstructured.Unstructured{}
	res.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "networking.k8s.io",
		Version: "v1",
		Kind:    "NetworkPolicy",
	})
	np, err := k8s.GetResource(res, "nsA", "npA")
	assert.NoError(t, err, "unable to get policy after rollback")
	assert.Equal(t, "", np.GetClusterName(),
		"Error (TestRollback): updated field should be empty after rollback")

	res = &unstructured.Unstructured{}
	res.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "crd.antrea.io",
		Version: "v1alpha1",
		Kind:    "NetworkPolicy",
	})
	_, err = k8s.GetResource(res, "nsA", "anpA")
	assert.NoError(t, err, "unable to get antrea policy after rollback")
}

func SetupMemRepo(storer *memory.Storage, fs billy.Filesystem) error {
	_, err := git.Init(storer, fs)
	fs.MkdirAll("k8s-policies", 0700)
	fs.MkdirAll("antrea-policies", 0700)
	fs.MkdirAll("antrea-cluster-policies", 0700)
	fs.MkdirAll("antrea-tiers", 0700)
	return err
}
