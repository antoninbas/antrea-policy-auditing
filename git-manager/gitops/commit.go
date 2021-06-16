package gitops

import (
    "fmt"
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

func AtomicAdd() (error) {
    r, err := git.PlainOpen(directory+"/network-policy-repository/")
    if err != nil {
        return err
    }
    w, err := r.Worktree()
    if err != nil {
        return err
    }

    Info("git add .")
    _, err = w.Add(".")
    return err
}

func AtomicCommit(username string, email string, message string) (error) {
    r, err := git.PlainOpen(directory+"/network-policy-repository/")
    if err != nil {
        return err
    }
    w, err := r.Worktree()
    if err != nil {
        return err
    }

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
    return directory+"/network-policy-repository/"+event.ObjectRef.Resource+"/"+event.ObjectRef.Namespace+"/"
}

func GetFileName(event auditv1.Event) (string) {
    return event.ObjectRef.Resource+event.ObjectRef.Namespace+event.ObjectRef.Name+".yaml"
}

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

func EventToDelete(event auditv1.Event) (error) {
    path := directory+"/network-policy-repository/"+event.ObjectRef.Resource+"/"+event.ObjectRef.Namespace+"/"
    path += event.ObjectRef.Resource+event.ObjectRef.Namespace+event.ObjectRef.Name+".yaml"

    err := os.Remove(path)
    return err
}

func HandleEventList(jsonstring string) (error) {
    eventList := auditv1.EventList{}
    err := json.Unmarshal(jsonstring, &eventList)
    if err != nil {
        return err
    }

    for _,event := range eventList.Items {
        switch verb := event.Verb; verb {
        case "create":
            ModifyFile(event)
        case "patch":
            ModifyFile(event)
        case "delete":
            EventToDelete(event)
        default:
            continue
        }
    }
}
