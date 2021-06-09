package main

import (
    "fmt"
    "log"
    "io/ioutil"
    "net/http"
)

//Simple webhook that reads and prints out incoming request bodies to stdout
func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}
    	fmt.Printf("%s\n", string(body))
    })
    fmt.Printf("Server listening on port 8080\n")
    if err := http.ListenAndServe(":8080", nil); err != nil {
	log.Fatal(err)
    }
}
