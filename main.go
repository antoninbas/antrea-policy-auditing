package main

import (
    "fmt"
    "flag"
    . "antrea-audit/git-manager/init"
    "antrea-audit/webhook"
)

func processArgs(portFlag *string, dirFlag *string) {
    flag.StringVar(portFlag, "p", "8080", "specifies port that audit webhook listens on, default 8080")
    flag.StringVar(dirFlag, "d", "", "path to which network policy repository is created, default current working directory")
    flag.Parse()
}

func main() {
    var (
        portFlag string
        dirFlag string
    )
    processArgs(&portFlag, &dirFlag)
    k8s, err := NewKubernetes()
    if err != nil {
            fmt.Println(err)
            return
    }
    if err := SetupRepo(k8s, &dirFlag); err != nil {
            fmt.Println(err)
            return
    }
    webhook.ReceiveEvents(dirFlag, portFlag)
}
