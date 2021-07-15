package test

import (
    "fmt"
    "testing"
    "antrea-audit/gitops"
    "antrea-audit/webhook"

    v1 "k8s.io/api/core/v1"
)

func TestExposeWebhook(t *testing.T) {
    mt := ""
    fakeK8sClient := NewK8sClientSet(Np1.inputResource)
	fakeCRDClient := NewCRDClientSet(Anp1.inputResource)
	k8s := &gitops.Kubernetes{
		PodCache:  map[string][]v1.Pod{},
		ClientSet: fakeK8sClient,
		CrdClient: fakeCRDClient,
	}

	cr, err := gitops.SetupRepo(k8s, "mem", mt)
	if err != nil {
		fmt.Println(err)
		t.Errorf("should not have error for correct file")
	}

    webhook.ReceiveEvents(mt, "8008", cr)
}
