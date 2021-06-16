package test

import (
    "fmt"
    "testing"
    "io/ioutil"

    "antrea-audit/git-manager/gitops"

    billy "github.com/go-git/go-billy/v5"
	memory "github.com/go-git/go-git/v5/storage/memory"
)

var directory string

func TestHandleEventList(t *testing.T) {
    jsonStr, err := ioutil.ReadFile("./files/audit-log.txt")

    err = gitops.HandleEventList(jsonStr)

    if err != nil {
        fmt.Println(err)
        t.Errorf("Error (TestHandleEventList): should not return error for correct event list")
    }
}

func SetupMemRepo(storer *memory.Storage, fs billy.Filesystem) (error) {
    os.Mkdir(directory + "/network-policy-repository/", 0700)
    r, err := git.Init(storer, fs)
}
