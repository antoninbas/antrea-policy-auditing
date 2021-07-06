package test

import (
    // "fmt"
    "time"
    "testing"
    "io/ioutil"

    . "antrea-audit/git-manager/init"
    "antrea-audit/git-manager/gitops"
    "github.com/go-git/go-git/v5"
    v1 "k8s.io/api/core/v1"
    memory "github.com/go-git/go-git/v5/storage/memory"
    memfs "github.com/go-git/go-billy/v5/memfs"
)

func TestFilterCommits(t *testing.T) {
    start := time.Now()
    time.Sleep(time.Millisecond*500)
    empty := ""
    storer := memory.NewStorage()
    fs := memfs.New()
    fakeK8sClient := newK8sClientSet(np1.inputResource)
	fakeCRDClient := newCRDClientSet(anp1.inputResource)
	k8s := &Kubernetes{
		PodCache:  map[string][]v1.Pod{},
		ClientSet: fakeK8sClient,
		CrdClient: fakeCRDClient,
	}

    if err := SetupRepoInMem(k8s, storer, fs); err != nil {
		t.Errorf("Error (TestFilterCommits): unable to set up repo for the first time")
	}
    jsonStr, err := ioutil.ReadFile("./files/audit-log.txt")
    if err != nil {
        t.Errorf("Error (TestFilterCommits): cannot read audit-log.txt")
    }
    r, err := git.Open(storer, fs)
    if err != nil {
        t.Errorf("Error (TestFilterCommits): cannot open in memory repo")
    }
    err = gitops.HandleEventListInMem(directory, r, fs, jsonStr)
    if err != nil {
        t.Errorf("Error (TestFilterCommits): cannot handle this event list")
    }
    until := time.Now()

    commits, err := gitops.FilterCommits(r, &empty, &start, &until, &empty)

    for _, c := range commits {
        if c.Author.Name == "audit-init" {
            continue
        }
        if c.Author.Name != "kubernetes-admin" {
            t.Errorf("Error (TestFilterCommits): incorrect commit author")
        }
        if c.Message == "" {
            t.Errorf("Error (TestFilterCommits): commit message empty")
        }
    }
}
