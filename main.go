package main

import (
	"flag"

	"antrea-audit/git-manager/client"
	. "antrea-audit/git-manager/init"
	"antrea-audit/webhook"

	"k8s.io/klog/v2"
)

func processArgs(portFlag *string, dirFlag *string) {
	flag.StringVar(portFlag, "p", "8080", "specifies port that audit webhook listens on")
	flag.StringVar(dirFlag, "d", "", "path to which network policy repository is created, default current working directory")
	flag.Parse()
}

func main() {
	var (
		portFlag string
		dirFlag  string
	)
	klog.InitFlags(nil)
	processArgs(&portFlag, &dirFlag)
	k8s, err := client.NewKubernetes()
	if err != nil {
		klog.ErrorS(err, "unable to create kube client")
		return
	}
	cr, err := SetupRepo(k8s, "disk", dirFlag)
	if err != nil {
		klog.ErrorS(err, "unable to set up network policy repository")
		return
	}
	if err := webhook.ReceiveEvents(dirFlag, portFlag, cr); err != nil {
		klog.ErrorS(err, "an error occurred while running the audit webhook service")
		return
	}
}
