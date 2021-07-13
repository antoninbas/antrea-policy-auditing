package test

import (
    // "fmt"
    "time"
    "testing"
    "io/ioutil"

    . "antrea-audit/gitops"
    v1 "k8s.io/api/core/v1"
)

func TestFilterCommits(t *testing.T) {
    start := time.Now()
    time.Sleep(time.Millisecond*500)
    empty := ""
    fakeK8sClient := NewK8sClientSet(Np1.inputResource)
	fakeCRDClient := NewCRDClientSet(Anp1.inputResource)
	k8s := &Kubernetes{
		PodCache:  map[string][]v1.Pod{},
		ClientSet: fakeK8sClient,
		CrdClient: fakeCRDClient,
	}

    jsonStr, err := ioutil.ReadFile("./files/audit-log.txt")
    if err != nil {
        t.Errorf("Error (TestFilterCommits): cannot read audit-log.txt")
    }
    cr, err := SetupRepo(k8s, "mem", empty)
    if err != nil {
		t.Errorf("Error (TestFilterCommits): unable to set up repo for the first time")
	}
    err = cr.HandleEventList(jsonStr)
    if err != nil {
        t.Errorf("Error (TestFilterCommits): cannot handle this event list")
    }
    until := time.Now()

    commits, err := cr.FilterCommits(&empty, &start, &until, &empty)

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
