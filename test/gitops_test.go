package test

import (
    "fmt"
    "testing"
    "io/ioutil"

    "antrea-audit/git-manager/gitops"
    "github.com/go-git/go-git/v5"

    billy "github.com/go-git/go-billy/v5"
    memory "github.com/go-git/go-git/v5/storage/memory"
    memfs "github.com/go-git/go-billy/v5/memfs"
)

var directory = ""

func TestHandleEventList(t *testing.T) {
    storer := memory.NewStorage()
    fs := memfs.New()

    err := SetupMemRepo(storer, fs)
    if err != nil {
        fmt.Println(err)
        t.Errorf("should not have error for correct file")
    }

    r, err := git.Open(storer, fs)
    if err != nil {
        fmt.Println(err)
        t.Errorf("should not have error for correct file")
    }

    jsonStr, err := ioutil.ReadFile("./files/audit-log.txt")
    if err != nil {
        fmt.Println(err)
        t.Errorf("should not have error for correct file")
    }

    err = gitops.HandleEventListInMem(directory, r, fs, jsonStr)
    if err != nil {
        fmt.Println(err)
        t.Errorf("should not have error for correct file")
    }
}

func SetupMemRepo(storer *memory.Storage, fs billy.Filesystem) (error) {
    _, err := git.Init(storer, fs)
    fs.MkdirAll("k8s-policies", 0700)
    fs.MkdirAll("antrea-policies", 0700)
    fs.MkdirAll("antrea-cluster-policies", 0700)
    return err
}
