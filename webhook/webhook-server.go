package webhook

import (
    "fmt"
    "log"
    "io/ioutil"
    "net/http"

    "antrea-audit/git-manager/gitops"
)

func ReceiveEvents(dir string, port string) {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        defer r.Body.Close()
        body, err := ioutil.ReadAll(r.Body)
        if err != nil {
            fmt.Println(err)
        }
        fmt.Printf("%s\n", string(body))
        if err := gitops.HandleEventList(dir, body); err != nil {
            fmt.Println(err)
        }
    })
    fmt.Println("Server listening on port", port)
    if err := http.ListenAndServe(":"+string(port), nil); err != nil {
	    log.Fatal(err)
    }
}
