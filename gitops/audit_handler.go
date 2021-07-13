package gitops

import (
    "bytes"
	"encoding/json"
    "k8s.io/klog/v2"

    auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
)

func (cr *CustomRepo) HandleEventList(jsonstring []byte) (error) {
    eventList := auditv1.EventList{}
    jsonstring = bytes.TrimPrefix(jsonstring, []byte("\xef\xbb\xbf"))
    err := json.Unmarshal(jsonstring, &eventList)
    if err != nil {
        klog.ErrorS(err, "unable to unmarshal json into event list struct")
        return err
    }
    cr.Mutex.Lock()
    defer cr.Mutex.Unlock()
    for _,event := range eventList.Items {
        if event.Stage != "ResponseComplete" || event.ResponseStatus.Status == "Failure" {
            klog.V(4).InfoS("Audit event skipped (audit Stage isn't ResponseComplete or audit has ResponseStatus failure)")
            continue
        }
        user := event.User.Username
        email := event.User.Username+"+"+event.User.UID+"@audit.antrea.io"
        message := resourceMap[event.ObjectRef.Resource+event.ObjectRef.APIGroup]+event.ObjectRef.Namespace+"/"+event.ObjectRef.Name
        switch verb := event.Verb; verb {
        case "create":
            if err := cr.modifyFile(event); err != nil {
                klog.ErrorS(err, "unable to create new resource")
                return err
            }
            if err := cr.AddAndCommit(user, email, "Created "+message); err != nil {
                klog.ErrorS(err, "unable to add/commit change")
                return err
            }
        case "patch":
            if err := cr.modifyFile(event); err != nil {
                klog.ErrorS(err, "unable to update resource")
                return err
            }
            if err := cr.AddAndCommit(user, email, "Updated "+message); err != nil {
                klog.ErrorS(err, "unable to add/commit change")
                return err
            }
        case "delete":
            if err := cr.deleteFile(event); err != nil {
                klog.ErrorS(err, "unable to delete resource")
                return err
            }
            if err := cr.AddAndCommit(user, email, "Deleted "+message); err != nil {
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
