package gitops

import (
	"strings"

	auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
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

func computePath(dir string, resource string, namespace string, file string) (string) {
	path := []string{dir, resource, namespace, file}
	return strings.Join(path, "/")
}

func getAbsRepoPath(dir string, event auditv1.Event) (string) {
	resource := dirMap[event.ObjectRef.Resource+event.ObjectRef.APIGroup]
	namespace := event.ObjectRef.Namespace
	return computePath(dir, resource, namespace, "")
}

func getRelRepoPath(event auditv1.Event) (string) {
	resource := dirMap[event.ObjectRef.Resource+event.ObjectRef.APIGroup]
	namespace := event.ObjectRef.Namespace
	path := computePath("", resource, namespace, "")
    return strings.TrimPrefix(path, "/")
}

func getFileName(event auditv1.Event) (string) {
    return event.ObjectRef.Name+".yaml"
}