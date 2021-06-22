package gitops

import (
    "os"
    // "fmt"
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

var dirMap = map[string]string{
    "networkpoliciesnetworking.k8s.io": "k8s-policies",
    "networkpoliciescrd.antrea.io": "antrea-policies",
    "clusternetworkpoliciescrd.antrea.io": "antrea-cluster-policies",
}

var resourceMap = map[string]string{
    "networkpoliciesnetworking.k8s.io": "K8s network policy ",
    "networkpoliciescrd.antrea.io": "Antrea network policy ",
    "clusternetworkpoliciescrd.antrea.io": "Antrea cluster network policy ",
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

func GetRepoPath(dir string, event auditv1.Event) (string) {
    return dir+"/"+dirMap[event.ObjectRef.Resource+event.ObjectRef.APIGroup]+"/"+event.ObjectRef.Namespace+"/"
}

func GetFileName(event auditv1.Event) (string) {
    return event.ObjectRef.Name+".yaml"
}

func ModifyFile(dir string, event auditv1.Event) (error) {
    y, err := yaml.JSONToYAML(event.ResponseObject.Raw)
    if err!=nil {
        return err
    }

    path := GetRepoPath(dir, event)
    if _, err := os.Stat(path); os.IsNotExist(err) {
        os.Mkdir(path, 0700)
    }
    path += GetFileName(event)

    err = ioutil.WriteFile(path, y, 0644)
    return err
}

func ModifyFileInMem(dir string, fs billy.Filesystem, event auditv1.Event) (error) {
    y, err := yaml.JSONToYAML(event.ResponseObject.Raw)
    if err != nil {
        return err
    }
    path := GetRepoPath(dir, event)+GetFileName(event)
    newfile, err := fs.Create(path)
    if err != nil {
        return err
    }
    newfile.Write(y)
    newfile.Close()
    return err
}

func EventToDelete(dir string, event auditv1.Event) (error) {
    err := os.Remove(GetRepoPath(dir, event)+GetFileName(event))
    return err
}

func EventToDeleteInMem(dir string, fs billy.Filesystem, event auditv1.Event) (error) {
    err := fs.Remove(GetRepoPath(dir, event)+GetFileName(event))
    return err
}

func HandleEventList(dir string, jsonstring []byte) (error) {
    eventList := auditv1.EventList{}
    err := json.Unmarshal(jsonstring, &eventList)
    if err != nil {
        return err
    }
    for _,event := range eventList.Items {
        if event.Stage != "ResponseComplete" || event.ResponseStatus.Status == "Failure" {
            continue
        }
        r, err := git.PlainOpen(dir)
        if err != nil {
            return err
        }
        user := event.User.Username
        email := event.User.Username+event.User.UID+"@audit.antrea.io"
        message := resourceMap[event.ObjectRef.Resource+event.ObjectRef.APIGroup]+event.ObjectRef.Name
        switch verb := event.Verb; verb {
        case "create":
            err = ModifyFile(dir, event)
            if err != nil {
                return err
            }
            AddAndCommit(r, user, email, "Created "+message)
        case "patch":
            err = ModifyFile(dir, event)
            if err != nil {
                return err
            }
            AddAndCommit(r, user, email, "Updated "+message)
        case "delete":
            err = EventToDelete(dir, event)
            if err != nil {
                return err
            }
            AddAndCommit(r, user, email, "Deleted "+message)
        default:
            continue
        }
    }

    return nil
}


func HandleEventListInMem(dir string, r *git.Repository, fs billy.Filesystem, jsonstring []byte) (error) {
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
        user := event.User.Username
        email := event.User.Username+event.User.UID+"@audit.antrea.io"
        message := resourceMap[event.ObjectRef.Resource+event.ObjectRef.APIGroup]+event.ObjectRef.Name
        switch verb := event.Verb; verb {
        case "create":
            err = ModifyFileInMem(dir, fs, event)
            if err != nil {
                return err
            }
            AddAndCommit(r, user, email, "Created "+message)
        case "patch":
            err = ModifyFileInMem(dir, fs, event)
            if err != nil {
                return err
            }
            AddAndCommit(r, user, email, "Updated "+message)
        case "delete":
            err = EventToDeleteInMem(dir, fs, event)
            if err != nil {
                return err
            }
            AddAndCommit(r, user, email, "Deleted "+message)
        default:
            continue
        }
    }
    return nil
}
