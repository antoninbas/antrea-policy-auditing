package test

import (
    "fmt"
    "testing"
    "os/exec"
)

func TestPVFail(t *testing.T) {
    count := 0
    addcmd := exec.Command("kubectl", "apply", "-f", "../test/files/antrea-resources.yaml")
    delcmd := exec.Command("kubectl", "delete", "-f", "../test/files/antrea-resources.yaml")
    for {
        err := addcmd.Run()
        if err != nil {
            fmt.Println(err.Error())
            t.Errorf("ligma")
        }
        err = delcmd.Run()
        if err != nil {
            fmt.Println(err.Error())
            t.Errorf("sugma")
        }
        count += 1
        fmt.Println(count)
    }
}
