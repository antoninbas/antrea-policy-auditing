package gitops

import (
	"os"
)

const (
	svcAcctNameEnvKey   = "SERVICEACCOUNT_NAME"
	svcAcctDefault      = "audit-account"
	podNamespaceEnvKey  = "POD_NAMESPACE"
	podNamespaceDefault = "default"
)

func GetAuditServiceAccount() string {
	svcAcctName := os.Getenv(svcAcctNameEnvKey)
	if svcAcctName == "" {
		svcAcctName = svcAcctDefault
	}
	return svcAcctName
}

func GetAuditPodNamespace() string {
	podNamespace := os.Getenv(podNamespaceEnvKey)
	if podNamespace == "" {
		podNamespace = podNamespaceDefault
	}
	return podNamespace
}
