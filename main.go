package main

import (
    "fmt"
    . "antrea-audit/git-manager/init"
)

func main() {
        k8s, err := NewKubernetes()
        if err != nil {
                fmt.Println(err)
        }
        // TODO: Process directory from flag
        var dir = ""
        if err := SetupRepo(k8s, dir); err != nil {
                fmt.Println(err)
        }
}
