/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package github

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func getClient(url string) *Client {
	return &Client{
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		base: url,
	}
}

func TestIsMember(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Bad method: %s", r.Method)
		}
		if r.URL.Path != "/orgs/k8s/members/person" {
			t.Errorf("Bad request path: %s", r.URL.Path)
		}
		http.Error(w, "204 No Content", http.StatusNoContent)
	}))
	defer ts.Close()
	c := getClient(ts.URL)
	mem, err := c.IsMember("k8s", "person")
	if err != nil {
		t.Errorf("Didn't expect error: %v", err)
	} else if !mem {
		t.Errorf("Should be member.")
	}
}

func TestCreateComment(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Bad method: %s", r.Method)
		}
		if r.URL.Path != "/repos/k8s/kuber/issues/5/comments" {
			t.Errorf("Bad request path: %s", r.URL.Path)
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Could not read request body: %v", err)
		}
		var ic IssueComment
		if err := json.Unmarshal(b, &ic); err != nil {
			t.Errorf("Could not unmarshal request: %v", err)
		} else if ic.Body != "hello" {
			t.Errorf("Wrong body: %s", ic.Body)
		}
		http.Error(w, "201 Created", http.StatusCreated)
	}))
	defer ts.Close()
	c := getClient(ts.URL)
	if err := c.CreateComment("k8s", "kuber", 5, "hello"); err != nil {
		t.Errorf("Didn't expect error: %v", err)
	}
}

func TestDeleteComment(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Bad method: %s", r.Method)
		}
		if r.URL.Path != "/repos/k8s/kuber/issues/comments/123" {
			t.Errorf("Bad request path: %s", r.URL.Path)
		}
		http.Error(w, "204 No Content", http.StatusNoContent)
	}))
	defer ts.Close()
	c := getClient(ts.URL)
	if err := c.DeleteComment("k8s", "kuber", 123); err != nil {
		t.Errorf("Didn't expect error: %v", err)
	}
}

func TestGetPullRequest(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Bad method: %s", r.Method)
		}
		if r.URL.Path != "/repos/k8s/kuber/pulls/12" {
			t.Errorf("Bad request path: %s", r.URL.Path)
		}
		pr := PullRequest{
			User: User{"bla"},
		}
		b, err := json.Marshal(&pr)
		if err != nil {
			t.Fatalf("Didn't expect error: %v", err)
		}
		fmt.Fprint(w, bytes.NewBuffer(b))
	}))
	defer ts.Close()
	c := getClient(ts.URL)
	pr, err := c.GetPullRequest("k8s", "kuber", 12)
	if err != nil {
		t.Errorf("Didn't expect error: %v", err)
	} else if pr.User.Login != "bla" {
		t.Errorf("Wrong user: %s", pr.User.Login)
	}
}

func TestCreateStatus(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Bad method: %s", r.Method)
		}
		if r.URL.Path != "/repos/k8s/kuber/statuses/abcdef" {
			t.Errorf("Bad request path: %s", r.URL.Path)
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Could not read request body: %v", err)
		}
		var s Status
		if err := json.Unmarshal(b, &s); err != nil {
			t.Errorf("Could not unmarshal request: %v", err)
		} else if s.Context != "c" {
			t.Errorf("Wrong context: %s", s.Context)
		}
		http.Error(w, "201 Created", http.StatusCreated)
	}))
	defer ts.Close()
	c := getClient(ts.URL)
	if err := c.CreateStatus("k8s", "kuber", "abcdef", Status{
		Context: "c",
	}); err != nil {
		t.Errorf("Didn't expect error: %v", err)
	}
}

func TestListIssueComments(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Bad method: %s", r.Method)
		}
		if r.URL.Path == "/repos/k8s/kuber/issues/15/comments" {
			ics := []IssueComment{{ID: 1}}
			b, err := json.Marshal(ics)
			if err != nil {
				t.Fatalf("Didn't expect error: %v", err)
			}
			w.Header().Set("Link", fmt.Sprintf(`<blorp>; rel="first", <https://%s/someotherpath>; rel="next"`, r.Host))
			fmt.Fprint(w, bytes.NewBuffer(b))
		} else if r.URL.Path == "/someotherpath" {
			ics := []IssueComment{{ID: 2}}
			b, err := json.Marshal(ics)
			if err != nil {
				t.Fatalf("Didn't expect error: %v", err)
			}
			fmt.Fprint(w, bytes.NewBuffer(b))
		} else {
			t.Errorf("Bad request path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()
	c := getClient(ts.URL)
	ics, err := c.ListIssueComments("k8s", "kuber", 15)
	if err != nil {
		t.Errorf("Didn't expect error: %v", err)
	} else if len(ics) != 2 {
		t.Errorf("Expected two issues, found %d: %v", len(ics), ics)
	} else if ics[0].ID != 1 || ics[1].ID != 2 {
		t.Errorf("Wrong issue IDs: %v", ics)
	}
}

func TestAddLabel(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Bad method: %s", r.Method)
		}
		if r.URL.Path != "/repos/k8s/kuber/issues/5/labels" {
			t.Errorf("Bad request path: %s", r.URL.Path)
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Could not read request body: %v", err)
		}
		var ls []string
		if err := json.Unmarshal(b, &ls); err != nil {
			t.Errorf("Could not unmarshal request: %v", err)
		} else if len(ls) != 1 {
			t.Errorf("Wrong length labels: %v", ls)
		} else if ls[0] != "yay" {
			t.Errorf("Wrong label: %s", ls[0])
		}
	}))
	defer ts.Close()
	c := getClient(ts.URL)
	if err := c.AddLabel("k8s", "kuber", 5, "yay"); err != nil {
		t.Errorf("Didn't expect error: %v", err)
	}
}

func TestRemoveLabel(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Bad method: %s", r.Method)
		}
		if r.URL.Path != "/repos/k8s/kuber/issues/5/labels/yay" {
			t.Errorf("Bad request path: %s", r.URL.Path)
		}
		http.Error(w, "204 No Content", http.StatusNoContent)
	}))
	defer ts.Close()
	c := getClient(ts.URL)
	if err := c.RemoveLabel("k8s", "kuber", 5, "yay"); err != nil {
		t.Errorf("Didn't expect error: %v", err)
	}
}
