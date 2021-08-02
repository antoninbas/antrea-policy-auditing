package main

import (
    "os"
    "fmt"
    "path"
    "net/http"
    "io/ioutil"

    "github.com/spf13/cobra"
)

var author string
var since string
var until string
var resource string
var namespace string
var name string

var commandName = path.Base(os.Args[0])

var rootCmd = &cobra.Command{
    Use:   commandName,
    Short: commandName + " is the command line tool for filtering commits",
    Long:  commandName + " is the command line tool for filtering commmits that supports post and get functionality",
}

func getURL() (string) {
    qstn := false
    flags := []string{author, since, until, resource, namespace, name}
    flagnames := []string{"author", "since", "until", "resource", "namespace", "name"}
    url := "http://localhost:8008/changes?"
    for f, flag := range flags {
        if flag != "" {
            if qstn {
                url += "&"
            }
            url += flagnames[f]+"="+flag
            qstn = true
        }
    }
    return url
}

var getCmd = &cobra.Command {
    Use: "get",
    Short: "get with file",
    Run: func(cmd *cobra.Command, args []string) {
        url := getURL()
        resp, err := http.Get(url)
        if err != nil {
            fmt.Println(err)
            return
        }
        defer resp.Body.Close()
        body, err :=  ioutil.ReadAll(resp.Body)
        if err != nil {
            fmt.Println(err)
            return
        }
        fmt.Println(string(body))
    },
}

func init() {
    getCmd.Flags().StringVarP(&author, "author", "a", "", "author of changes")
    getCmd.Flags().StringVarP(&since, "since", "s", "", "start of time range")
    getCmd.Flags().StringVarP(&until, "until", "u", "", "end of time range")
    getCmd.Flags().StringVarP(&resource, "resource", "r", "", "resource nameto filter commits by")
    getCmd.Flags().StringVarP(&namespace, "namespace", "p", "", "namespace to filter commits by")
    getCmd.Flags().StringVarP(&name, "name", "n", "", "name to filter commits by")
    rootCmd.AddCommand(getCmd)
}

func main() {
    err := rootCmd.Execute()
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
