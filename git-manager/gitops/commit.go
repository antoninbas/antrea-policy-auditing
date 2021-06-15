package gitops

import (
    "fmt"
    "os"
    "time"
    "io/ioutil"

    "github.com/go-git/go-git/v5"
    . "github.com/go-git/go-git/v5/_examples"
    "github.com/go-git/go-git/v5/plumbing/object"
    "github.com/ghodss/yaml"

    auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
)

var directory string

func EventToCommit(event auditv1.Event) (error) {
    r, err := git.PlainOpen(directory+"/network-policy-repository/")
    if err != nil {
        return err
    }
    w, err := r.Worktree()

    Info("git add .")
    _, err = w.Add(".")
    if err != nil {
        return err
    }

    Info("git commit -m \"Network Policy change commit\"")
    _, err = w.Commit("Network Policy change commit", &git.CommitOptions{
        Author: &object.Signature{
            Name:  event.User.Username,
            Email: event.User.Username+event.User.UID+"@audit.antrea.io",
            When:  time.Now(),
        },
    })
    if err != nil {
        return err
    }

    return nil
}
func EventListToCommit(eventList auditv1.EventList) (error) {
    for _,event := range eventList.Items {
        err := EventToCommit(event)
        if err != nil {
            return err
        }
    }
    return nil
}

func ModifyFile(event auditv1.Event) (error) {
    y, err := yaml.JSONToYAML(event.ResponseObject.Raw)
    if err!=nil {
        fmt.Printf("error converting json to yaml\n")
        return err
    }

    path := directory+"/network-policy-repository/"+event.ObjectRef.Resource+"/"+event.ObjectRef.Namespace+"/"
    if _, err := os.Stat(path); os.IsNotExist(err) {
        os.Mkdir(path, 0700)
    }

    path += event.ObjectRef.Resource+event.ObjectRef.Namespace+event.ObjectRef.Name+".yaml"
    err = ioutil.WriteFile(path, y, 0644)
    if err!=nil {
        fmt.Printf("error writing file\n")
        return err
    }

    return nil
}

func ModifyFiles(eventList auditv1.EventList) (error) {
    for _,event := range eventList.Items {
        err := ModifyFile(event)
        if err != nil {
            return err
        }
    }
    return nil
}
