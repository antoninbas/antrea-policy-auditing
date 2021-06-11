package main

import (
    "github.com/go-git/go-git/v5"
    . "github.com/go-git/go-git/v5/_examples"
)

func main() {
    url := "https://github.com/Dhruv-J/shiny-sniffle"
    directory := "./shiny-sniffle/"

    Info("git clone %s %s --recursive", url, directory)
    
    _, err := git.PlainClone(directory, false, &git.CloneOptions{
		URL:               url,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})

	CheckIfError(err)
}
