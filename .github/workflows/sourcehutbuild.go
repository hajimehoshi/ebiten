// Copyright 2021 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	flagCommit   = flag.String("commit", "", "git commit hash ID")
	flagManifest = flag.String("manifest", "", "manifest file path")
	flagNote     = flag.String("note", "", "note for the build")
)

func init() {
	flag.Parse()
}

var secret string

func init() {
	secret = os.Getenv("SOURCEHUT_OAUTH_TOKEN")
}

type JobRequest struct {
	Manifest string   `json:"manifest"`
	Note     string   `json:"note,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Execute  *bool    `json:"execute,omitempty"`
	Secrets  *bool    `json:"secrets,omitempty"`
}

type JobResponse struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

func httpRequest(method string, path string, body interface{}) (io.ReadCloser, error) {
	const baseURL = `https://builds.sr.ht`

	var reqBody io.Reader
	if body != nil {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		if err := enc.Encode(body); err != nil {
			return nil, err
		}
		reqBody = &buf
	}

	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+secret)
	if reqBody != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if got, want := res.StatusCode, http.StatusOK; got != want {
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("status code must be %d but %d; URL: %s, response: %s", want, got, req.URL.String(), string(resBody))
	}

	return res.Body, nil
}

func run() error {
	manifest, err := os.ReadFile(*flagManifest)
	if err != nil {
		return err
	}

	manifestStr := strings.ReplaceAll(string(manifest), "{{.Commit}}", *flagCommit)

	body, err := httpRequest(http.MethodPost, "/api/jobs", &JobRequest{
		Manifest: manifestStr,
		Note:     *flagNote,
	})
	if err != nil {
		return err
	}
	defer body.Close()

	var jobRes JobResponse
	dec := json.NewDecoder(body)
	if err := dec.Decode(&jobRes); err != nil {
		return err
	}
	fmt.Printf("Job ID: %d\n", jobRes.ID)

	// Poll the queued job's status
	const maxAttempt = 60
	for i := 0; i < maxAttempt; i++ {
		fmt.Printf("Polling the status... (%d)\n", i+1)

		body, err := httpRequest(http.MethodGet, fmt.Sprintf("/api/jobs/%d", jobRes.ID), nil)
		if err != nil {
			return err
		}

		var jobRes JobResponse
		dec := json.NewDecoder(body)
		if err := dec.Decode(&jobRes); err != nil {
			return err
		}

		switch jobRes.Status {
		case "pending":
			// Do nothing
		case "queued":
			// Do nothing
		case "running":
			// Do nothing
		case "success":
			return nil
		case "failed":
			resBody, err := io.ReadAll(body)
			if err != nil {
				return err
			}
			return fmt.Errorf("failed; response: %s", resBody)
		}

		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("time out")
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
