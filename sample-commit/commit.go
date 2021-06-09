package main

import (
    "fmt"
    "os"
    "time"
    "math/rand"
    "io/ioutil"

    "github.com/go-git/go-git/v5"
    . "github.com/go-git/go-git/v5/_examples"
    "github.com/go-git/go-git/v5/plumbing/object"
    "gopkg.in/yaml.v2"

    auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
    authv1 "k8s.io/api/authentication/v1"
    // "k8s.io/apimachinery/pkg/runtime"
)

var directory string

func setupRepository() {
    if directory == "" {
        path, err := os.Getwd()
        if err != nil {
            return
        }
        directory = path
    }
    os.Mkdir(directory+"network-policy-repository", 0700)
    r, err := git.PlainInit(directory+"/network-policy-repository/", false)
    CheckIfError(err)
    w, err := r.Worktree()
    CheckIfError(err)

    os.Mkdir(directory+"/network-policy-repository/k8s-policy", 0700)
    os.Mkdir(directory+"/network-policy-repository/antrea-policy", 0700)
    os.Mkdir(directory+"/network-policy-repository/antrea-cluster-policy", 0700)

    Info("git add .")
    _, err = w.Add(".")
    CheckIfError(err)

    Info("git commit -m \"test commit number 1a\"")
    _, err = w.Commit("test commit number 1a", &git.CommitOptions{
        Author: &object.Signature{
            Name:  "John Doe",
            Email: "john@doe.org",
            When:  time.Now(),
        },
    })
    CheckIfError(err)
}

func eventToCommit(event auditv1.Event) {
    r, err := git.PlainOpen(directory+"/network-policy-repository/")
    CheckIfError(err)
    w, err := r.Worktree()
    CheckIfError(err)

    Info("git add .")
    _, err = w.Add(".")
    CheckIfError(err)

    Info("git commit -m \"Network Policy change commit\"")
    _, err = w.Commit("Network Policy change commit", &git.CommitOptions{
        Author: &object.Signature{
            Name:  event.User.Username,
            Email: "example@example.com",
            When:  time.Now(),
        },
    })
    CheckIfError(err)
}

func modifyFile(event auditv1.Event) {
    requestobj := event.RequestObject
    if requestobj==nil {
        return
    }
    d, err := yaml.Marshal(&requestobj)
    if err!=nil {
        fmt.Println("error marshalling")
        return
    }
    path := directory + "/network-policy-repository/"
    err = ioutil.WriteFile(path, d, 0644)
    if err!=nil {
        fmt.Println("error writing file")
        return
    }
}

func main() {
    setupRepository()

    user0 := authv1.UserInfo{
        Username: "user0",
    }
    user1 := authv1.UserInfo{
        Username: "user1",
    }
    user2 := authv1.UserInfo{
        Username: "user2",
    }
    event := auditv1.Event{}

    for i:=0; i<5; i++ {
        n := rand.Intn(3)
        if n==0 {
            event.User = user0
            eventToCommit(event)
        } else if n==1 {
            event.User = user1
            eventToCommit(event)
        } else {
            event.User = user2
            eventToCommit(event)
        }
    }

    modifyFile(event)
}
