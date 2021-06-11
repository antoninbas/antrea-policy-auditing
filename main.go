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

        if err := SetupRepo(k8s); err != nil {
                fmt.Println(err)
        }
}
