package test

import (
	"antrea-audit/gitops"
	"antrea-audit/webhook"
	"fmt"
	"testing"
)

func TestExposeWebhook(t *testing.T) {
    mt := ""
	fakeClient := NewClient(Np1.inputResource, Anp1.inputResource)
	k8s := &gitops.K8sClient{
		Client: fakeClient,
	}

	cr, err := gitops.SetupRepo(k8s, gitops.StorageModeInMemory, mt)
	if err != nil {
		fmt.Println(err)
		t.Errorf("should not have error for correct file")
	}

    webhook.ReceiveEvents("8008", cr)
}
