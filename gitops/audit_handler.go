package gitops

import (
	"bytes"
	"encoding/json"
	"fmt"

	auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
	"k8s.io/klog/v2"
)

func (cr *CustomRepo) HandleEventList(jsonstring []byte) error {
	cr.Mutex.Lock()
	defer cr.Mutex.Unlock()
	eventList := auditv1.EventList{}
	jsonstring = bytes.TrimPrefix(jsonstring, []byte("\xef\xbb\xbf"))
	err := json.Unmarshal(jsonstring, &eventList)
	if err != nil {
		return fmt.Errorf("could not unmarshal event list json: %w", err)
	}
	for _, event := range eventList.Items {
		if event.Stage != "ResponseComplete" ||
			event.ResponseStatus.Status == "Failure" ||
			event.User.Username == cr.ServiceAccount {
			klog.V(2).InfoS("audit event skipped (audit Stage != ResponseComplete, audit ResponseStatus != Success, or audit produced by rollback)")
			continue
		}
		if cr.RollbackMode {
			return fmt.Errorf("rollback in progress")
		}
		if err = cr.HandleEvent(event); err != nil {
			return fmt.Errorf("could not handle event: %w", err)
		}
	}
	return nil
}

func (cr *CustomRepo) HandleEvent(event auditv1.Event) error {
	user := event.User.Username
	email := event.User.Username + "+" + event.User.UID + "@audit.antrea.io"
	message := resourceMap[event.ObjectRef.Resource+event.ObjectRef.APIGroup] + event.ObjectRef.Namespace + "/" + event.ObjectRef.Name
	switch verb := event.Verb; verb {
	case "create":
		if err := cr.modifyFile(event); err != nil {
			return fmt.Errorf("could not create new resource: %w", err)
		}
		if err := cr.AddAndCommit(user, email, "Created "+message); err != nil {
			return fmt.Errorf("could not add/commit add operation: %w", err)
		}
		klog.V(2).InfoS("successfully created resource", "resource", message)
	case "patch":
		if err := cr.modifyFile(event); err != nil {
			return fmt.Errorf("could not update resource: %w", err)
		}
		if err := cr.AddAndCommit(user, email, "Updated "+message); err != nil {
			return fmt.Errorf("could not add/commit patch operation: %w", err)
		}
		klog.V(2).InfoS("successfully updated resource", "resource", message)
	case "delete":
		if err := cr.deleteFile(event); err != nil {
			return fmt.Errorf("could not delete resource: %w", err)
		}
		if err := cr.AddAndCommit(user, email, "Deleted "+message); err != nil {
			return fmt.Errorf("could not add/commit the delete operation: %w", err)
		}
		klog.V(2).InfoS("successfully deleted resource", "resource", message)
	default:
		return fmt.Errorf("must be create/patch/delete operation")
	}
	return nil
}
