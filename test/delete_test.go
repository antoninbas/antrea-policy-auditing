package main

import (
    "fmt"
    "testing"
    "io/ioutil"
    "encoding/json"

    "antrea-audit/git-manager/gitops"

    auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
)

func TestEventToDelete(t *testing.T) {
    jsonStr, _ := ioutil.ReadFile("./files/delete-log.txt")
    eventList := auditv1.EventList{}
    err := json.Unmarshal(jsonStr, &eventList)

    err = gitops.EventToDelete(eventList.Items[0])

    if err != nil {
        fmt.Println(err)
        t.Errorf("Error (TestEventToDelete): should not return error for correct event list")
    }
}
