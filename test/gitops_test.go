package test

import (
	"fmt"
	"io/ioutil"
	"testing"

	"antrea-audit/gitops"

	"github.com/go-git/go-git/v5"

	billy "github.com/go-git/go-billy/v5"
	memory "github.com/go-git/go-git/v5/storage/memory"

	v1 "k8s.io/api/core/v1"
)

var directory = ""

func TestHandleEventList(t *testing.T) {
	fakeK8sClient := NewK8sClientSet(Np1.inputResource)
	fakeCRDClient := NewCRDClientSet(Anp1.inputResource)
	k8s := &gitops.Kubernetes{
		PodCache:  map[string][]v1.Pod{},
		ClientSet: fakeK8sClient,
		CrdClient: fakeCRDClient,
	}

	jsonStr, err := ioutil.ReadFile("./files/audit-log.txt")
	if err != nil {
		fmt.Println(err)
		t.Errorf("should not have error for correct file")
	}

	cr, err := gitops.SetupRepo(k8s, "mem", directory)
	if err != nil {
		fmt.Println(err)
		t.Errorf("should not have error for correct file")
	}

	err = cr.HandleEventList(jsonStr)
	if err != nil {
		fmt.Println(err)
		t.Errorf("should not have error for correct file")
	}
}

// func TestTagging(t *testing.T) {
// 	storer := memory.NewStorage()
// 	fs := memfs.New()

// 	err := SetupMemRepo(storer, fs)
// 	if err != nil {
// 		fmt.Println(err)
// 		t.Errorf("Error (TestTagging): unable to set up repo in memory properly")
// 	}
// 	r, err := git.Open(storer, fs)
// 	if err != nil {
// 		t.Errorf("Error (TestTagging): unable to open in mem repo")
// 	}
// 	if err := gitops.AddAndCommit(r, "test-user", "test@antrea.audit.io", "dummy commit"); err != nil {
// 		t.Errorf("Error (TestTagging): unable to create dummy commit")
// 	}
// 	h, err := r.Head()
// 	if err != nil {
// 		t.Errorf("Error (TestTagging): unable to get repo head ref")
// 	}

// 	testSig := &object.Signature{
// 		Name:  "test",
// 		Email: "test@antrea.audit.io",
// 		When:  time.Now(),
// 	}
// 	// Attempt to add tag to nonexistent commit
// 	if err := gitops.TagCommit(r, "bad-hash", "test-tag", testSig); err == nil {
// 		t.Errorf("Error (TestTagging): should have returned error on bad commit hash")
// 	}

// 	// Create new tags successfully
// 	if err := gitops.TagCommit(r, h.Hash().String(), "test-tag", testSig); err != nil {
// 		t.Errorf("Error (TestTagging): unable to create new tag")
// 	}
// 	if err := gitops.TagCommit(r, h.Hash().String(), "test-tag-2", testSig); err != nil {
// 		t.Errorf("Error (TestTagging): unable to create new tag")
// 	}
// 	_, err = r.Tag("test-tag")
// 	if err != nil {
// 		t.Errorf("Error (TestTagging): could not retrieve created tag")
// 	}
// 	_, err = r.Tag("test-tag-2")
// 	if err != nil {
// 		t.Errorf("Error (TestTagging): could not retrieve created tag")
// 	}

// 	// Attempt to add tag with the same name
// 	if err := gitops.TagCommit(r, h.Hash().String(), "test-tag", testSig); err != nil {
// 		t.Errorf("Error (TestTagging): unable to handle duplicate tag creation")
// 	}
// 	tags, _ := r.TagObjects()
// 	count := 0
// 	if err := tags.ForEach(func(tag *object.Tag) error {
// 		count += 1
// 		return nil
// 	}); err != nil {
// 		t.Errorf("Error (TestTagging): could not iterate through repo tags")
// 	}
// 	assert.Equal(t, count, 2, "Error (TestTagging): duplicate tag detected, tag count should have been 2")
// }

func SetupMemRepo(storer *memory.Storage, fs billy.Filesystem) error {
	_, err := git.Init(storer, fs)
	fs.MkdirAll("k8s-policies", 0700)
	fs.MkdirAll("antrea-policies", 0700)
	fs.MkdirAll("antrea-cluster-policies", 0700)
	return err
}
