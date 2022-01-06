/*
Copyright © 2022 Pavel Tisnovsky

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

package main

import (
	"fmt"
	"os"
	"strings"

	"encoding/base64"
	"encoding/json"
	"github.com/verdverm/frisby"
	"net/http"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	server "github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// common constants used by REST API tests
const (
	apiURL              = "http://localhost:8080/api/v1/"
	contentTypeHeader   = "Content-Type"
	contentLengthHeader = "Content-Length"

	authHeaderName = "x-rh-identity"

	// ContentTypeJSON represents MIME type for JSON format
	ContentTypeJSON = "application/json; charset=utf-8"

	// ContentTypeJSON represents MIME type for JSON format
	ContentTypeJSONWithoutCharset = "application/json"

	// ContentTypeText represents MIME type for plain text format
	ContentTypeText = "text/plain; charset=utf-8"

	// knownOrganizationID represents ID of known organization
	knownOrganizationID = "1"

	// unknownOrganizationID represents ID of inknown organization
	unknownOrganizationID = "100000"

	wrongOrganizationID = "foobar"

	knownClusterForOrganization1   = "00000000-0000-0000-0000-000000000000"
	knownCluster2ForOrganization1  = "00000000-0000-0000-ffff-000000000000"
	knownCluster3ForOrganization1  = "00000000-0000-0000-0000-ffffffffffff"
	unknownClusterForOrganization1 = "00000000-0000-0000-0000-000000000001"

	// no response status
	None = ""
)

// states
const (
	OkStatusResponse = server.OkStatusPayload
	MissingAuthToken = "Missing auth token"
)

// list of known organizations that are stored in test database
var knownOrganizations = []int{1, 2, 3, 4}

// list of unknown organizations that are not stored in test database
var unknownOrganizations = []int{5, 6, 7, 8}

// list of improper organization IDs
var improperOrganizations = []int{-1000, -1, 0}

// user account number
const accountNumber = "42"

// setAuthHeaderForOrganization set authorization header to request
func setAuthHeaderForOrganization(f *frisby.Frisby, orgID int) {
	plainHeader := fmt.Sprintf("{\"identity\": {\"internal\": {\"org_id\": \"%d\"}, \"account_number\":\"%s\"}}", orgID, accountNumber)
	encodedHeader := base64.StdEncoding.EncodeToString([]byte(plainHeader))
	f.SetHeader(authHeaderName, encodedHeader)
}

// setAuthHeader set authorization header to request for organization 1
func setAuthHeader(f *frisby.Frisby) {
	setAuthHeaderForOrganization(f, 1)
}

// constructURLForReportForOrgCluster function constructs an URL to access the
// endpoint to retrieve results for given cluster from selected organization
func constructURLForReportForOrgCluster(organizationID string, clusterID string, userID types.UserID) string {
	url := httputils.MakeURLToEndpoint("", server.ReportEndpoint, organizationID, clusterID, userID)
	return url[1:]
}

// constructURLForReportInfoForOrgCluster function constructs an URL to access
// the endpoint to retrieve results metadata for given cluster from selected
// organization
func constructURLForReportInfoForOrgCluster(organizationID string,
	clusterID string, userID types.UserID) string {
	url := httputils.MakeURLToEndpoint("", server.ReportMetainfoEndpoint,
		organizationID, clusterID, userID)
	return url[1:]
}

// readStatusFromResponse reads and parses status from response body
func readStatusFromResponse(f *frisby.Frisby) StatusOnlyResponse {
	response := StatusOnlyResponse{}
	text, err := f.Resp.Content()
	if err != nil {
		f.AddError(err.Error())
	} else {
		err := json.Unmarshal(text, &response)
		if err != nil {
			f.AddError(err.Error())
		}
	}
	return response
}

// statusResponseChecker tests which text is returned in "status" attribute
func statusResponseChecker(f *frisby.Frisby, expectedStatus string) {
	response := readStatusFromResponse(f)
	if response.Status != expectedStatus {
		f.AddError(fmt.Sprintf("Expected status is '%s', but got '%s' instead", expectedStatus, response.Status))
	}
}

// StatusOnlyResponse represents response containing just a status
type StatusOnlyResponse struct {
	Status string `json:"status"`
}

// ClustersResponse represents response containing list of clusters for given
// organization
type ClustersResponse struct {
	Clusters []string `json:"clusters"`
	Status   string   `json:"status"`
}

// InfoResponse represents response from /info endpoint
type InfoResponse struct {
	Info   map[string]string `json:"info"`
	Status string            `json:"status"`
}

func metricsEndPointContentTypeChecker(f *frisby.Frisby) {
	f.Expect(func(f *frisby.Frisby) (bool, string) {
		header := f.Resp.Header.Get(contentTypeHeader)
		if strings.HasPrefix(header, "text/plain") {
			return true, OkStatusResponse
		}
		return false, fmt.Sprintf("Expected Header %q to be %q, but got %q", contentTypeHeader, "text/plain", header)
	})
}

// elementary checks for /info endpoint
func infoResponseChecker(f *frisby.Frisby) {
	var expectedInfoKeys []string = []string{
		"BuildBranch",
		"BuildCommit",
		"BuildTime",
		"BuildVersion",
		"DB_version",
		"UtilsVersion",
	}

	// check the response
	text, err := f.Resp.Content()
	if err != nil {
		f.AddError(err.Error())
	} else {
		response := InfoResponse{}
		err := json.Unmarshal(text, &response)
		if err != nil {
			f.AddError(err.Error())
		}
		if response.Status != "ok" {
			f.AddError("Expecting 'status' to be set to 'ok'")
		}
		if len(response.Info) == 0 {
			f.AddError("Info node is empty")
		}
		for _, expectedKey := range expectedInfoKeys {
			_, found := response.Info[expectedKey]
			if !found {
				f.AddError("Info node does not contain key " + expectedKey)
			}
		}
	}
}

// RestAPITest represents specification of one REST API call (request) and
// expected response
type RestAPITest struct {
	Endpoint               string
	Method                 string
	Message                string
	AuthHeader             bool
	AuthHeaderOrganization int
	ExpectedStatus         int
	ExpectedContentType    string
	ExpectedResponseStatus string
	AdditionalChecker      func(F *frisby.Frisby)
}

func main() {
	fmt.Println("vim-go")

}
