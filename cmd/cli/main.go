package main

import (
    "os"
    "fmt"
    "path"
    "net/http"
    "io/ioutil"
    "encoding/json"

    "antrea-audit/webhook"
    "github.com/spf13/cobra"
)

var commandName = path.Base(os.Args[0])

var rootCmd = &cobra.Command{
    Use:   commandName,
    Short: commandName + " is the command line tool for filtering commits",
    Long:  commandName + " is the command line tool for filtering commmits that supports post and get functionality",
}

func getURL(requestBody []byte) (string) {
    filts := webhook.Filters{}
    err := json.Unmarshal(requestBody, &filts)
    if err != nil {
        fmt.Println(err)
        return ""
    }
    qstn := false
    url := "http://localhost:8008/changes?"
    if filts.Author != "" {
        url += "author="+filts.Author
        qstn = true
    }
    if !filts.Since.IsZero() {
        if qstn {
            url += "&"
        }
        url += "since="+filts.Since.String()
        qstn = true
    }
    if !filts.Until.IsZero() {
        if qstn {
            url += "&"
        }
        url += "until="+filts.Until.String()
        qstn = true
    }
    if filts.FileName != "" {
        if qstn {
            url += "&"
        }
        url += "filename="+filts.FileName
    }
    return url
}

var getCmd = &cobra.Command {
    Use: "get",
    Short: "get with file",
    Args: cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        requestBody, err := ioutil.ReadFile(args[0])
        if err != nil {
            fmt.Println(err)
            return
        }

        url := getURL(requestBody)
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
    rootCmd.AddCommand(getCmd)
}

func main() {
    err := rootCmd.Execute()
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}
