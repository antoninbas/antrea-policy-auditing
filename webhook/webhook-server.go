package webhook

import (
    "fmt"
    "log"
    "io/ioutil"
    "net/http"

    "antrea-audit/git-manager/gitops"
)

func ReceiveEvents() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        defer r.Body.Close()
        body, err := ioutil.ReadAll(r.Body)
        if err != nil {
            return
        }
        fmt.Printf("%s\n", string(body))
        gitops.HandleEventList(body)
    })
    fmt.Printf("Server listening on port 8080\n")
    if err := http.ListenAndServe(":8080", nil); err != nil {
	    log.Fatal(err)
    }
}
