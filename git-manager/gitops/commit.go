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
    "k8s.io/klog/v2"
    billy "github.com/go-git/go-billy/v5"
)

var dirMap = map[string]string{
    "networkpoliciesnetworking.k8s.io": "k8s-policies",
    "networkpoliciescrd.antrea.io": "antrea-policies",
    "clusternetworkpoliciescrd.antrea.io": "antrea-cluster-policies",
    "tierscrd.antrea.io": "antrea-tiers",
}

var resourceMap = map[string]string{
    "networkpoliciesnetworking.k8s.io": "K8s network policy ",
    "networkpoliciescrd.antrea.io": "Antrea network policy ",
    "clusternetworkpoliciescrd.antrea.io": "Antrea cluster network policy ",
    "tierscrd.antrea.io": "Antrea tier ",
}

func AddAndCommit(r *git.Repository, username string, email string, message string) (error) {
    w, err := r.Worktree()
    if err != nil {
        klog.ErrorS(err, "unable to get git worktree from repository")
        return err
    }
    _, err = w.Add(".")
    if err != nil {
        klog.ErrorS(err, "unable to add git change to worktree")
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
        klog.ErrorS(err, "unable to commit git change to worktree")
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
    if err != nil {
        klog.ErrorS(err, "unable to convert event ResponseObject from JSON to YAML format")
        return err
    }
    path := GetRepoPath(dir, event)
    if _, err := os.Stat(path); os.IsNotExist(err) {
        os.Mkdir(path, 0700)
    }
    path += GetFileName(event)
    if err := ioutil.WriteFile(path, y, 0644); err != nil {
        klog.ErrorS(err, "unable to write/update file in repository")
        return err
    }
    return nil
}

func ModifyFileInMem(dir string, fs billy.Filesystem, event auditv1.Event) (error) {
    y, err := yaml.JSONToYAML(event.ResponseObject.Raw)
    if err != nil {
        klog.ErrorS(err, "unable to convert event ResponseObject from JSON to YAML format")
        return err
    }
    path := GetRepoPath(dir, event)+GetFileName(event)
    newfile, err := fs.Create(path)
    if err != nil {
        klog.ErrorS(err, "unable to create file at: ", "path", path)
        return err
    }
    newfile.Write(y)
    newfile.Close()
    return nil
}

func EventToDelete(dir string, event auditv1.Event) (error) {
    path := GetRepoPath(dir, event) + GetFileName(event)
    if err := os.Remove(path); err != nil {
        klog.ErrorS(err, "unable to remove file at: ", "path", path)
        return err
    }
    return nil
}

func EventToDeleteInMem(dir string, fs billy.Filesystem, event auditv1.Event) (error) {
    path := GetRepoPath(dir, event) + GetFileName(event)
    if err := fs.Remove(path); err != nil {
        klog.ErrorS(err, "unable to remove file at: ", "path", path)
        return err
    }
    return nil
}

func HandleEventList(dir string, jsonstring []byte) (error) {
    eventList := auditv1.EventList{}
    err := json.Unmarshal(jsonstring, &eventList)
    if err != nil {
        klog.ErrorS(err, "unable to unmarshal json into event list struct")
        return err
    }
    for _,event := range eventList.Items {
        if event.Stage != "ResponseComplete" || event.ResponseStatus.Status == "Failure" {
            klog.V(4).InfoS("Audit event skipped (audit Stage isn't ResponseComplete or audit has ResponseStatus failure)")
            continue
        }
        r, err := git.PlainOpen(dir)
        if err != nil {
            klog.ErrorS(err, "unable to open repository")
            return err
        }
        user := event.User.Username
        email := event.User.Username+"+"+event.User.UID+"@audit.antrea.io"
        message := resourceMap[event.ObjectRef.Resource+event.ObjectRef.APIGroup]+event.ObjectRef.Name
        switch verb := event.Verb; verb {
        case "create":
            if err := ModifyFile(dir, event); err != nil {
                klog.ErrorS(err, "unable to create new resource")
                return err                
            }
            if err := AddAndCommit(r, user, email, "Created "+message); err != nil {
                klog.ErrorS(err, "unable to add/commit change")
                return err
            }
        case "patch":
            if err := ModifyFile(dir, event); err != nil {
                klog.ErrorS(err, "unable to update resource")
                return err
            }
            if err := AddAndCommit(r, user, email, "Updated "+message); err != nil {
                klog.ErrorS(err, "unable to add/commit change")
                return err  
            }
        case "delete":
            if err := EventToDelete(dir, event); err != nil {
                klog.ErrorS(err, "unable to delete resource")
                return err
            }
            if err := AddAndCommit(r, user, email, "Deleted "+message); err != nil {
                klog.ErrorS(err, "unable to add/commit change")
                return err  
            }
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
        klog.ErrorS(err, "unable to open repository")
        return err
    }

    for _,event := range eventList.Items {
        if event.Stage != "ResponseComplete" || event.ResponseStatus.Status == "Failure" {
            continue
        }
        user := event.User.Username
        email := event.User.Username+"+"+event.User.UID+"@audit.antrea.io"
        message := resourceMap[event.ObjectRef.Resource+event.ObjectRef.APIGroup]+event.ObjectRef.Name
        switch verb := event.Verb; verb {
        case "create":
            if err := ModifyFileInMem(dir, fs, event); err != nil {
                klog.ErrorS(err, "unable to create new resource")
                return err                
            }
            if err := AddAndCommit(r, user, email, "Created "+message); err != nil {
                klog.ErrorS(err, "unable to add/commit change")
                return err
            }
        case "patch":
            if err = ModifyFileInMem(dir, fs, event); err != nil {
                klog.ErrorS(err, "unable to update resource")
                return err
            }
            if err := AddAndCommit(r, user, email, "Updated "+message); err != nil {
                klog.ErrorS(err, "unable to add/commit change")
                return err  
            }
        case "delete":
            if err = EventToDeleteInMem(dir, fs, event); err != nil {
                klog.ErrorS(err, "unable to delete resource")
                return err
            }
            if err := AddAndCommit(r, user, email, "Deleted "+message); err != nil {
                klog.ErrorS(err, "unable to add/commit change")
                return err  
            }
        default:
            continue
        }
    }
    return nil
}
