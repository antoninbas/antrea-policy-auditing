package gitops

import (
    "os"

    auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
)

func EventToDelete(event auditv1.Event) (error) {
    path := directory+"/network-policy-repository/"+event.ObjectRef.Resource+"/"+event.ObjectRef.Namespace+"/"
    path += event.ObjectRef.Resource+event.ObjectRef.Namespace+event.ObjectRef.Name+".yaml"

    err := os.Remove(path)
    return err
}
