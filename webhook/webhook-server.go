package webhook

import (
    "fmt"
    "log"
    "io/ioutil"
    "net/http"

    "antrea-audit/git-manager/gitops"
)

func ReceiveEvents(port string) {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        defer r.Body.Close()
        body, err := ioutil.ReadAll(r.Body)
        if err != nil {
            return
        }
        fmt.Printf("%s\n", string(body))
        gitops.HandleEventList(body)
    })
    fmt.Println("Server listening on port", port)
    if err := http.ListenAndServe(":"+string(port), nil); err != nil {
	    log.Fatal(err)
    }
}
