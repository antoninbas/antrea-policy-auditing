package gitops

import (
	"bytes"
	"encoding/json"

	"k8s.io/klog/v2"

	auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
)

func (cr *CustomRepo) HandleEventList(jsonstring []byte) error {
	// cr.Mutex.Lock()
	// defer cr.Mutex.Unlock()
	eventList := auditv1.EventList{}
	jsonstring = bytes.TrimPrefix(jsonstring, []byte("\xef\xbb\xbf"))
	err := json.Unmarshal(jsonstring, &eventList)
	if err != nil {
		klog.ErrorS(err, "unable to unmarshal json into event list struct")
		return err
	}
	for _, event := range eventList.Items {
		if event.Stage != "ResponseComplete" ||
			event.ResponseStatus.Status == "Failure" ||
			event.User.Username == cr.ServiceAccount {
			klog.V(4).InfoS("Audit event skipped (audit Stage != ResponseComplete, audit ResponseStatus != Success, or audit produced by rollback)")
			continue
		}
		// if cr.RollbackMode {
		// 	klog.V(2).Infof("Rollback currently in progress, rejecting audit")
		// 	return fmt.Errorf("rollback-in-progress")
		// }
		user := event.User.Username
		email := event.User.Username + "+" + event.User.UID + "@audit.antrea.io"
		message := resourceMap[event.ObjectRef.Resource+event.ObjectRef.APIGroup] + event.ObjectRef.Namespace + "/" + event.ObjectRef.Name
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
			klog.V(2).Infof("Successfully updated resource: %s", message)
		case "delete":
			if err := cr.deleteFile(event); err != nil {
				klog.ErrorS(err, "unable to delete resource")
				return err
			}
			if err := cr.AddAndCommit(user, email, "Deleted "+message); err != nil {
				klog.ErrorS(err, "unable to add/commit change")
				return err
			}
			klog.V(2).Infof("Successfully deleted resource: %s", message)
		default:
			continue
		}
	}
	return nil
}
