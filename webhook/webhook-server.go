package webhook

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"antrea-audit/gitops"

	"k8s.io/klog/v2"
	
	//"github.com/go-git/go-git/v5/plumbing/object"
)

type Change struct {
	Sha     string `json:"sha"`
	Author  string `json:"author"`
	Message string `json:"Message"`
}

type filters struct {
	Author   string    `json:"author"`
	Since    time.Time `json:"since"`
	Until    time.Time `json:"until"`
	FileName string    `json:"filename"`
}

type rollbackRequest struct {
	Tag string
	//TargetCommit *object.Commit `json:"commit"`
}

type tagRequest struct {
	Tag string
	Sha string
	//Signature *object.Signature
}

func events(w http.ResponseWriter, r *http.Request, cr *gitops.CustomRepo) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.ErrorS(err, "unable to read audit body")
		w.WriteHeader(http.StatusBadRequest)
	}
	klog.V(3).Infof("Audit received: %s", string(body))
	if err := cr.HandleEventList(body); err != nil {
		if err.Error() == "rollback-in-progress" {
			klog.ErrorS(err, "audit received during rollback")
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			klog.ErrorS(err, "unable to process audit event list")
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func changes(w http.ResponseWriter, r *http.Request, cr *gitops.CustomRepo) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.ErrorS(err, "unable to read audit body")
		w.WriteHeader(http.StatusBadRequest)
	}
	klog.V(3).Infof("Filters received: %s", string(body))
	filts := filters{}
	if err := json.Unmarshal(body, &filts); err != nil {
		klog.ErrorS(err, "unable to marshal request body")
		w.WriteHeader(http.StatusBadRequest)
	}
	commits, err := cr.FilterCommits(&filts.Author, &filts.Since, &filts.Until, &filts.FileName)
	if err != nil {
		klog.ErrorS(err, "unable to process audit event list")
		w.WriteHeader(http.StatusBadRequest)
	}
	var changes []Change
	for _, c := range commits {
		chg := Change{}
		chg.Sha = c.Hash.String()
		chg.Author = c.Author.Name
		chg.Message = c.Message
		changes = append(changes, chg)
	}
	jsonstring, err := json.Marshal(changes)
	if err != nil {
		klog.ErrorS(err, "unable to marshal list of changes")
		w.WriteHeader(http.StatusInternalServerError)
	}
	_, err = w.Write(jsonstring)
	if err != nil {
		klog.ErrorS(err, "unable to write json to response writer")
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func rollback(w http.ResponseWriter, r *http.Request, cr *gitops.CustomRepo) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.ErrorS(err, "unable to read audit body")
		w.WriteHeader(http.StatusBadRequest)
	}
	rollbackRequest := rollbackRequest{}
	if err := json.Unmarshal(body, &rollbackRequest); err != nil {
		klog.ErrorS(err, "unable to marshal request body")
		w.WriteHeader(http.StatusBadRequest)
	}
	// commit, _ := cr.TagToCommit(rollbackRequest.Tag)
	// if err := cr.RollbackRepo(commit); err != nil {
	// 	klog.ErrorS(err, "failed to rollback repo")
	// 	w.WriteHeader(http.StatusInternalServerError)
	// }
}

func tag(w http.ResponseWriter, r *http.Request, cr *gitops.CustomRepo) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.ErrorS(err, "unable to read audit body")
		w.WriteHeader(http.StatusBadRequest)
	}
	tagRequest := tagRequest{}
	if err := json.Unmarshal(body, &tagRequest); err != nil {
		klog.ErrorS(err, "unable to marshal request body")
		w.WriteHeader(http.StatusBadRequest)
	}
	// if err := cr.TagCommit(tagRequest.Sha, tagRequest.Tag, tagRequest.Signature); err != nil {
	// 	klog.ErrorS(err, "failed to tag commit")
	// 	w.WriteHeader(http.StatusInternalServerError)
	// }
}

func ReceiveEvents(port string, cr *gitops.CustomRepo) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		events(w, r, cr)
	})
	http.HandleFunc("/changes", func(w http.ResponseWriter, r *http.Request) {
		changes(w, r, cr)
	})
	http.HandleFunc("/rollback", func(w http.ResponseWriter, r *http.Request) {
		rollback(w, r, cr)
	})
	klog.V(2).Infof("Audit webhook server started, listening on port %s", port)
	if err := http.ListenAndServe(":"+string(port), nil); err != nil {
		klog.ErrorS(err, "Audit webhook service died")
		return err
	}
	return nil
}
