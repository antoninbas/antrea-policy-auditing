package gitops

import (
    "os"
    "time"
    "io/ioutil"
    "encoding/json"

    "github.com/go-git/go-git/v5"
    . "github.com/go-git/go-git/v5/_examples"
    "github.com/go-git/go-git/v5/plumbing/object"
    "github.com/ghodss/yaml"

    auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
)

var directory string
var dirmap = map[string]string{
    "networkpoliciesnetworking.k8s.io": "k8s-policy",
    "networkpoliciescrd.antrea.io": "antrea-policy",
    "clusternetworkpoliciescrd.antrea.io": "antrea-cluster-policy",
}

func AddAndCommit(r *git.Repository, username string, email string, message string) (error) {
    w, err := r.Worktree()
    if err != nil {
        return err
    }

    Info("git add .")
    _, err = w.Add(".")
    return err

    Info("git commit -m \""+message+"\"")
    _, err = w.Commit(message, &git.CommitOptions{
        Author: &object.Signature{
            Name: username,
            Email: email,
            When: time.Now(),
        },
    })
    return err
}

func GetRepoPath(event auditv1.Event) (string) {
    return directory+"/network-policy-repository/"+dirmap[event.ObjectRef.Resource+event.ObjectRef.APIGroup]+"/"+event.ObjectRef.Namespace+"/"
}

func GetFileName(event auditv1.Event) (string) {
    return ObjectRef.Name+".yaml"
}

func EventToCommit(event auditv1.Event) (error) {
    r, err := git.PlainOpen(directory+"/network-policy-repository/")
    if err != nil {
        return err
    }
    return AddAndCommit(r, event.User.Username, event.User.Username+event.User.UID+"@audit.antrea.io", "Network Policy Change for file: "+GetFileName(event))
}

func ModifyFile(event auditv1.Event) (error) {
    y, err := yaml.JSONToYAML(event.ResponseObject.Raw)
    if err!=nil {
        return err
    }

    path := GetRepoPath(event)
    if _, err := os.Stat(path); os.IsNotExist(err) {
        os.Mkdir(path, 0700)
    }
    path += GetFileName(event)

    err = ioutil.WriteFile(path, y, 0644)
    return err
}

func EventToDelete(event auditv1.Event) (error) {
    err := os.Remove(GetRepoPath(event)+GetFileName(event))
    return err
}

func HandleEventList(jsonstring []byte) (error) {
    eventList := auditv1.EventList{}
    err := json.Unmarshal(jsonstring, &eventList)
    if err != nil {
        return err
    }

    for _,event := range eventList.Items {
        switch verb := event.Verb; verb {
        case "create":
            err = ModifyFile(event)
            if err != nil {
                return err
            }
        case "patch":
            err = ModifyFile(event)
            if err != nil {
                return err
            }
        case "delete":
            err = EventToDelete(event)
            if err != nil {
                return err
            }
        default:
            continue
        }
        err = EventToCommit(event)
        if err != nil {
            return err
        }
    }

    return nil
}
