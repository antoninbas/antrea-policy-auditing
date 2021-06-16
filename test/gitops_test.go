package test

import (
    "fmt"
    "testing"
    "io/ioutil"

    "antrea-audit/git-manager/gitops"
)

func TestHandleEventList(t *testing.T) {
    jsonStr, err := ioutil.ReadFile("./files/audit-log.txt")

    err = gitops.HandleEventList(jsonStr)

    if err != nil {
        fmt.Println(err)
        t.Errorf("Error (TestHandleEventList): should not return error for correct event list")
    }
}
