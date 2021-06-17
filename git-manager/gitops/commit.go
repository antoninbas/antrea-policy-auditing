package gitops

import (
    "os"
    "time"
    "bytes"
    "io/ioutil"
    "encoding/json"

    "github.com/go-git/go-git/v5"
    "github.com/go-git/go-git/v5/plumbing/object"
    "github.com/ghodss/yaml"

    auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
    billy "github.com/go-git/go-billy/v5"
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

    _, err = w.Add(".")
    if err != nil {
        return err
    }

    _, err = w.Commit(message, &git.CommitOptions{
        Author: &object.Signature{
            Name: username,
            Email: email,
            When: time.Now(),
        },
    })
    if err != nil {
        return err
    }
    return nil
}

func GetRepoPath(event auditv1.Event) (string) {
    return directory+"/network-policy-repository/"+dirmap[event.ObjectRef.Resource+event.ObjectRef.APIGroup]+"/"+event.ObjectRef.Namespace+"/"
}

func GetFileName(event auditv1.Event) (string) {
    return event.ObjectRef.Name+".yaml"
}

func EventToCommit(r *git.Repository, event auditv1.Event) (error) {
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

func ModifyFileInMem(fs billy.Filesystem, event auditv1.Event) (error) {
    y, err := yaml.JSONToYAML(event.ResponseObject.Raw)
    if err != nil {
        return err
    }

    path := GetRepoPath(event)+GetFileName(event)
    newfile, err := fs.Create(path)
    if err != nil {
        return err
    }
    newfile.Write(y)
    newfile.Close()
    return err
}

func EventToDelete(event auditv1.Event) (error) {
    err := os.Remove(GetRepoPath(event)+GetFileName(event))
    return err
}

func EventToDeleteInMem(fs billy.Filesystem, event auditv1.Event) (error) {
    err := fs.Remove(GetRepoPath(event)+GetFileName(event))
    return err
}

func HandleEventList(jsonstring []byte) (error) {
    eventList := auditv1.EventList{}
    err := json.Unmarshal(jsonstring, &eventList)
    if err != nil {
        return err
    }

    r, err := git.PlainOpen(directory+"/network-policy-repository/")
    if err != nil {
        return err
    }

    for _,event := range eventList.Items {
        if event.Stage != "ResponseComplete" || event.ResponseStatus.Status == "Failure" {
            continue
        }
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
        err = EventToCommit(r, event)
        if err != nil {
            return err
        }
    }

    return nil
}


func HandleEventListInMem(r *git.Repository, fs billy.Filesystem, jsonstring []byte) (error) {
    eventList := auditv1.EventList{}
    jsonstring = bytes.TrimPrefix(jsonstring, []byte("\xef\xbb\xbf"))
    err := json.Unmarshal(jsonstring, &eventList)
    if err != nil {
        return err
    }

    for _,event := range eventList.Items {
        if event.Stage != "ResponseComplete" || event.ResponseStatus.Status == "Failure" {
            continue
        }
        switch verb := event.Verb; verb {
        case "create":
            err = ModifyFileInMem(fs, event)
            if err != nil {
                return err
            }
        case "patch":
            err = ModifyFileInMem(fs, event)
            if err != nil {
                return err
            }
        case "delete":
            err = EventToDeleteInMem(fs, event)
            if err != nil {
                return err
            }
        default:
            continue
        }
        err = EventToCommit(r, event)
    }

    return nil
}
