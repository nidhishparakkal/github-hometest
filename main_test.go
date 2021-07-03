package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func Test_postError(t *testing.T) {
	tests := []struct {
		name string
		code int
		want string
	}{
		{name: "test-200-OK", code: http.StatusOK, want: "OK\n"},
		{name: "test-400-Bad-Request", code: http.StatusBadRequest, want: "Bad Request\n"},
		{name: "test-500-Internal-Server-Error", code: http.StatusInternalServerError, want: "Internal Server Error\n"},
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		t.Run(tt.name, func(t *testing.T) {
			postError(w, tt.code)
		})
		got := w.Body.String()
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("hookHandler() = %v, want %v", got, tt.want)
		}
	}
}

func Test_createBranchProtection(t *testing.T) {
	type branchProtectionInput struct {
		RequiredStatusChecks struct {
			Strict   bool          `json:"strict"`
			Contexts []interface{} `json:"contexts"`
		} `json:"required_status_checks"`
		EnforceAdmins              bool `json:"enforce_admins"`
		RequiredPullRequestReviews struct {
			RequireCodeOwnerReviews      bool `json:"require_code_owner_reviews"`
			RequiredApprovingReviewCount int  `json:"required_approving_review_count"`
		} `json:"required_pull_request_reviews"`
		Restrictions                   interface{} `json:"restrictions"`
		RequiredConversationResolution bool        `json:"required_conversation_resolution"`
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if !reflect.DeepEqual(r.Method, http.MethodPut) {
			t.Errorf("createIssue() = %v, want %v", r.Method, http.MethodPut)
		}
		body, _ := ioutil.ReadAll(r.Body)
		var got branchProtectionInput
		var want branchProtectionInput
		json.Unmarshal(body, &got)
		input, _ := ioutil.ReadFile("./payload_branch_protection.json")
		json.Unmarshal(input, &want)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("createIssue() = %v, want %v", got, want)
		}
	}))
	defer ts.Close()
	client := http.Client{}
	tests := []struct {
		name   string
		url    string
		token  string
		client *http.Client
		want   int
	}{
		{
			name: "test-branch-protection-200OK", url: ts.URL, token: "test-token", client: &client, want: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, _ := createBranchProtection(tt.url, tt.token, tt.client)
			got := resp.StatusCode
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createBranchProtection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createIssue(t *testing.T) {
	type issueInput struct {
		Title  string   `json:"title"`
		Labels []string `json:"labels"`
		Body   string   `json:"body"`
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if !reflect.DeepEqual(r.Method, http.MethodPost) {
			t.Errorf("createIssue() = %v, want %v", r.Method, http.MethodPost)
		}
		body, _ := ioutil.ReadAll(r.Body)
		var got issueInput
		var want issueInput
		json.Unmarshal(body, &got)
		input, _ := ioutil.ReadFile("./payload_create_issue.json")
		json.Unmarshal(input, &want)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("createIssue() = %v, want %v", got, want)
		}
	}))
	defer ts.Close()
	client := http.Client{}
	tests := []struct {
		name   string
		url    string
		token  string
		client *http.Client
		want   int
	}{
		{
			name: "test-issue-create-200OK", url: ts.URL, token: "test-token", client: &client, want: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, _ := createIssue(tt.url, tt.token, tt.client)
			got := resp.StatusCode
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createIssue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_rootHandler(t *testing.T) {
	type testData struct {
		name    string
		request *http.Request
		want    string
	}
	tests := []testData{
		testData{name: "test-wrong-root-path", request: httptest.NewRequest("GET", "/test", nil), want: "Path not found"},
		testData{name: "test-correct-root-path", request: httptest.NewRequest("GET", "/", nil), want: "<html><head><title>Github hometest API</title></head><body><h1>Github hometest API</h1></body></html>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			rootHandler(w, tt.request)
			got := w.Body.String()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rootHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hookHandler(t *testing.T) {
	type testData struct {
		name    string
		request *http.Request
		want    string
	}
	emptyRepoMsg, _ := json.Marshal(postMsg{Action: "created", Repository: postRepo{FullName: ""}})
	tests := []testData{
		testData{name: "test-get-request", request: httptest.NewRequest("GET", "/webhook", nil), want: "ok"},
		testData{name: "test-post-empty-body", request: httptest.NewRequest("POST", "/webhook", ioutil.NopCloser(bytes.NewBufferString(""))), want: "Bad Request\n"},
		testData{name: "test-post-empty-repo", request: httptest.NewRequest("POST", "/webhook", ioutil.NopCloser(bytes.NewBuffer(emptyRepoMsg))), want: "No Content\n"},
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		t.Run(tt.name, func(t *testing.T) {
			hookHandler(w, tt.request)
			got := w.Body.String()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("hookHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}
