package main

import (
	"flag"

	"antrea-audit/gitops"
	"antrea-audit/webhook"

	"k8s.io/klog/v2"
)

func processArgs() {
	flag.StringVar(&portFlag, "p", "8080", "specifies port that audit webhook listens on")
	flag.StringVar(&dirFlag, "d", "", "directory where resource repository is created, defaults to current working directory")
	flag.Parse()
}

var (
	portFlag string
	dirFlag  string
)

func main() {
	klog.InitFlags(nil)
	processArgs()
	k8s, err := gitops.NewKubernetes()
	if err != nil {
		klog.ErrorS(err, "unable to create kube client")
		return
	}
	cr, err := gitops.SetupRepo(k8s, gitops.StorageModeDisk, dirFlag)
	if err != nil {
		klog.ErrorS(err, "unable to set up resource repository")
		return
	}
	if err := webhook.ReceiveEvents(portFlag, cr); err != nil {
		klog.ErrorS(err, "an error occurred while running the audit webhook service")
		return
	}
}
