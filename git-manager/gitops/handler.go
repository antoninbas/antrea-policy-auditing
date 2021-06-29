package gitops

import (
	"bytes"
	"encoding/json"

    "github.com/go-git/go-git/v5"

    auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
    "k8s.io/klog/v2"
    billy "github.com/go-git/go-billy/v5"
)

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
        message := resourceMap[event.ObjectRef.Resource+event.ObjectRef.APIGroup]+event.ObjectRef.Namespace+"/"+event.ObjectRef.Name
        switch verb := event.Verb; verb {
        case "create":
            if err := modifyFile(dir, event); err != nil {
                klog.ErrorS(err, "unable to create new resource")
                return err                
            }
            if err := AddAndCommit(r, user, email, "Created "+message); err != nil {
                klog.ErrorS(err, "unable to add/commit change")
                return err
            }
        case "patch":
            if err := modifyFile(dir, event); err != nil {
                klog.ErrorS(err, "unable to update resource")
                return err
            }
            if err := AddAndCommit(r, user, email, "Updated "+message); err != nil {
                klog.ErrorS(err, "unable to add/commit change")
                return err  
            }
        case "delete":
            if err := deleteFile(r, dir, event); err != nil {
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
        klog.V(2).Infof("Successfully updated resource: %s", message)
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
            if err := modifyFileInMem(dir, fs, event); err != nil {
                klog.ErrorS(err, "unable to create new resource")
                return err                
            }
            if err := AddAndCommit(r, user, email, "Created "+message); err != nil {
                klog.ErrorS(err, "unable to add/commit change")
                return err
            }
        case "patch":
            if err = modifyFileInMem(dir, fs, event); err != nil {
                klog.ErrorS(err, "unable to update resource")
                return err
            }
            if err := AddAndCommit(r, user, email, "Updated "+message); err != nil {
                klog.ErrorS(err, "unable to add/commit change")
                return err  
            }
        case "delete":
            if err := deleteFile(r, dir, event); err != nil {
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