package test

import (
	"antrea-audit/gitops"
	"antrea-audit/webhook"
	"fmt"
	"io/ioutil"
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
    jsonStr, err := ioutil.ReadFile("./files/audit-log.txt")
    if err != nil {
        fmt.Println(err)
        t.Errorf("could not read audit-log file")
    }
    err = cr.HandleEventList(jsonStr)
    if err != nil {
        fmt.Println(err)
        t.Errorf("could not handle audit event list")
    }

    webhook.ReceiveEvents("8008", cr)
}
