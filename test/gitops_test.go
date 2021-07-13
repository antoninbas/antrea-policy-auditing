package test

import (
    "fmt"
    "testing"
    "io/ioutil"

    "antrea-audit/gitops"

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
