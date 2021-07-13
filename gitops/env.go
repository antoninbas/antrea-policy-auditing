package gitops

import (
	"os"

	"k8s.io/klog"
)

const (
	svcAcctNameEnvKey  = "SERVICEACCOUNT_NAME"
	podNamespaceEnvKey = "POD_NAMESPACE"
)

func GetAuditServiceAccount() string {
	svcAcctName := os.Getenv(svcAcctNameEnvKey)
	if svcAcctName == "" {
		svcAcctName = "antrea-audit"
	}
	return svcAcctName
}

func GetPodNamespace() string {
	podNamespace := os.Getenv(podNamespaceEnvKey)
	if podNamespace == "" {
		klog.Warningf("Environment variable %s not found", podNamespaceEnvKey)
	}
	return podNamespace
}
