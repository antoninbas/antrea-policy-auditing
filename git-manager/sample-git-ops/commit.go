package main

import (
    "fmt"
    "time"

    "github.com/go-git/go-git/v5"
    . "github.com/go-git/go-git/v5/_examples"
    "github.com/go-git/go-git/v5/plumbing/object"
)

func main() {
    directory := "./shiny-sniffle/"
    r, err := git.PlainOpen(directory)
	CheckIfError(err)
	w, err := r.Worktree()
	CheckIfError(err)

    Info("git add README.md")
	_, err = w.Add("README.md")
	CheckIfError(err)

    Info("git status --porcelain")
	status, err := w.Status()
	CheckIfError(err)
    fmt.Println(status)

    Info("git commit -m \"test commit number 1a\"")
	_, err = w.Commit("test commit number 1a", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "John Doe",
			Email: "john@doe.org",
			When:  time.Now(),
		},
	})
	CheckIfError(err)
}
