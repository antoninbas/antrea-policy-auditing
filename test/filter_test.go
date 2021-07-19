package test

import (
	"io/ioutil"
	"testing"
	"time"

	. "antrea-audit/gitops"
)

func TestFilterCommits(t *testing.T) {
	start := time.Now()
	time.Sleep(time.Millisecond * 500)
	empty := ""
	fakeK8sClient := NewK8sClientSet(Np1.inputResource)
	fakeCRDClient := NewCRDClientSet(Anp1.inputResource)
	k8s := &KubeClients{
		ClientSet: fakeK8sClient,
		CrdClient: fakeCRDClient,
	}

	jsonStr, err := ioutil.ReadFile("./files/audit-log.txt")
	if err != nil {
		t.Errorf("Error (TestFilterCommits): cannot read audit-log.txt")
	}
	cr, err := SetupRepo(k8s, StorageModeInMemory, empty)
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
