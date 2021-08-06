package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"antrea-audit/types"

	"github.com/spf13/cobra"
)

// get changes flags
var author, since, until, resource, namespace, name string

// tag flags
var tagAuthor, tagEmail string

// rollback flags
var rollbackTag, rollbackSHA string

var commandName = path.Base(os.Args[0])

var rootCmd = &cobra.Command{
	Use:  commandName,
	Long: commandName + " is the command line tool for managing the auditing resource repository",
}

const port = "8080"

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "get with file",
	Run: func(cmd *cobra.Command, args []string) {
		url := getURL()
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(body))
	},
}

var tagCmd = &cobra.Command{
	Use:   "tag create tag_name commit_sha [-a author] [-e email]\n   or: tag delete tag_name",
	Short: "tags commits in the repository",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("too few args")
		}
		if args[0] == "create" {
			if len(args) != 3 {
				return fmt.Errorf("unexpected number of args for tag create")
			}
		} else if args[0] == "delete" {
			if len(args) != 2 {
				return fmt.Errorf("unexpected number of args for tag delete")
			}
		} else {
			return fmt.Errorf("unsupported keyword (not create or delete)")
		}
		return nil
	},
	Run: runTag,
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback -t tag_name | -s commit_sha",
	Short: "rollback to the specified commit by tag name or SHA",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("unexpected number of args for rollback")
		}
		if (rollbackTag != "") == (rollbackSHA != "") {
			return fmt.Errorf("must specify exactly one of -t or -s")
		}
		return nil
	},
	Run: runRollback,
}

func getURL() string {
	flags := []string{author, since, until, resource, namespace, name}
	flagnames := []string{"author=", "since=", "until=", "resource=", "namespace=", "name="}
	var parts []string
	for i, flag := range flags {
		if strings.TrimSpace(flag) != "" {
			parts = append(parts, flagnames[i]+flag)
		}
	}
	url := "http://localhost:" + port + "/changes?"
	url += strings.Join(parts, "&")
	return url
}

func runTag(cmd *cobra.Command, args []string) {
	url := "http://localhost:" + port + "/tag"
	var request types.TagRequest
	if args[0] == "create" {
		request = types.TagRequest{
			Type:   types.TagCreate,
			Tag:    args[1],
			Sha:    args[2],
			Author: tagAuthor,
			Email:  tagEmail,
		}
	} else {
		request = types.TagRequest{
			Type: types.TagDelete,
			Tag:  args[1],
		}
	}
	j, err := json.Marshal(request)
	if err != nil {
		fmt.Println(err)
		return
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(j))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusInternalServerError {
		fmt.Println("Error encountered while processing tag request")
		return
	}
	fmt.Println(string(body))
}

func runRollback(cmd *cobra.Command, args []string) {
	url := "http://localhost:" + port + "/rollback"
	request := types.RollbackRequest{
		Tag: rollbackTag,
		Sha: rollbackSHA,
	}
	j, err := json.Marshal(request)
	if err != nil {
		fmt.Println(err)
		return
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(j))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusInternalServerError {
		fmt.Println("Error encountered while processing rollback request")
		return
	}
	fmt.Println(string(body))
}

func init() {
	getCmd.Flags().StringVarP(&author, "author", "a", "", "author of changes")
	getCmd.Flags().StringVarP(&since, "since", "s", "", "start of time range")
	getCmd.Flags().StringVarP(&until, "until", "u", "", "end of time range")
	getCmd.Flags().StringVarP(&resource, "resource", "r", "", "resource nameto filter commits by")
	getCmd.Flags().StringVarP(&namespace, "namespace", "p", "", "namespace to filter commits by")
	getCmd.Flags().StringVarP(&name, "name", "n", "", "name to filter commits by")
	rootCmd.AddCommand(getCmd)
	tagCmd.Flags().StringVarP(&tagAuthor, "author", "a", "no-author", "tag author")
	tagCmd.Flags().StringVarP(&tagEmail, "email", "e", "default@audit.io", "tag email")
	rootCmd.AddCommand(tagCmd)
	rollbackCmd.Flags().StringVarP(&rollbackTag, "tag", "t", "", "name of tag")
	rollbackCmd.Flags().StringVarP(&rollbackSHA, "SHA", "s", "", "commit hash to rollback to")
	rootCmd.AddCommand(rollbackCmd)
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
