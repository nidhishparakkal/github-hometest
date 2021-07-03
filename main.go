package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type postRepo struct {
	FullName string `json:"full_name"`
}

type postMsg struct {
	Action     string `json:"action"`
	Repository postRepo
}

func postError(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}

func createBranchProtection(url string, token string, client *http.Client) (*http.Response, error) {
	body, err := ioutil.ReadFile("./payload_branch_protection.json")
	if err != nil {
		return nil, errors.New("unable to load file")
	}
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.New("unable to create new http request")
	}
	req.Header.Add("Accept", "application/vnd.github.luke-cage-preview+json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("token %s", token))

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("unable to process PUT reqeust")
	}
	return resp, nil
}

func createIssue(url string, token string, client *http.Client) (*http.Response, error) {
	body, err := ioutil.ReadFile("./payload_create_issue.json")
	if err != nil {
		return nil, errors.New("unable to load file")
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.New("unable to create new http request")
	}
	req.Header.Add("Accept", "application/vnd.github.v3+json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("token %s", token))

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("unable to process POST reqeust")
	}
	return resp, nil
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Path not found"))
		return
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><head><title>Github hometest API</title></head><body><h1>Github hometest API</h1></body></html>`))
	}
}

func hookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if r.Body == nil {
			postError(w, http.StatusNoContent)
			return
		}
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			postError(w, http.StatusBadRequest)
			return
		}
		jsonBody := postMsg{}
		if err := json.Unmarshal(body, &jsonBody); err != nil {
			postError(w, http.StatusBadRequest)
			return
		}
		if jsonBody.Action == "created" {
			token, _ := os.LookupEnv("TOKEN")
			repo := jsonBody.Repository.FullName
			if repo == "" {
				postError(w, http.StatusNoContent)
				return
			}
			branch := "main"
			url := fmt.Sprintf("https://api.github.com/repos/%s/branches/%s/protection", repo, branch)
			client := http.Client{}
			resp, err := createBranchProtection(url, token, &client)
			if err != nil {
				fmt.Println(err.Error())
			}
			if resp.StatusCode == http.StatusOK {
				fmt.Printf("Branch protection rule created on [%s] for [%s] branch...\n", repo, branch)
			} else {
				fmt.Printf("Error creating branch protection rule! Response code: %d\n", resp.StatusCode)
			}

			url = fmt.Sprintf("https://api.github.com/repos/%s/issues", repo)
			resp, err = createIssue(url, token, &client)
			if err != nil {
				fmt.Println(err.Error())
			}
			if resp.StatusCode == http.StatusCreated {
				fmt.Printf("New issue created on [%s]...\n", repo)
			} else {
				fmt.Printf("Error creating issue! Response code: %d\n", resp.StatusCode)
			}
		} else {
			postError(w, http.StatusNotAcceptable)
			return
		}
	} else if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		return
	} else {
		postError(w, http.StatusMethodNotAllowed)
		return
	}
}

func main() {
	fmt.Println("Starting hometest api v1...")
	if _, ok := os.LookupEnv("TOKEN"); !ok {
		fmt.Println("Error!! Token missing")
		os.Exit(1)
	}
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "80"
	}
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/webhook", hookHandler)
	fmt.Println(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", port), nil))
	os.Exit(1)
}
