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

// checkEndPoint performs request to selected endpoint and check the response
func checkEndPoint(test *RestAPITest) {
	// prepare Frisby test object
	url := apiURL + test.Endpoint
	f := frisby.Create(test.Message)
	f.Method = test.Method
	f.Url = url

	if test.AuthHeader {
		if test.AuthHeaderOrganization != 0 {
			setAuthHeaderForOrganization(f, test.AuthHeaderOrganization)
		} else {
			setAuthHeader(f)
		}
	}

	// perform the request
	f.Send()

	// check the response
	f.ExpectStatus(test.ExpectedStatus)

	// check the response type
	if test.ExpectedContentType != None {
		f.ExpectHeader(contentTypeHeader, test.ExpectedContentType)
	}

	// perform additional check, if setup
	if test.AdditionalChecker != nil {
		test.AdditionalChecker(f)
	}

	// status can be returned in JSON format too
	if test.ExpectedResponseStatus != None {
		statusResponseChecker(f, test.ExpectedResponseStatus)
	}

	// print overall status of test to terminal
	f.PrintReport()
}

// runAllTests function run all REST API tests provided in argument. Number of
// errors found is returned (zero in case of no error).
func runAllTests(tests []RestAPITest) int {
	for _, test := range tests {
		checkEndPoint(&test)
	}
	frisby.Global.PrintReport()
	return frisby.Global.NumErrored
}

var tests []RestAPITest = []RestAPITest{
	{
		Message:                "Check the entry point to REST API using HTTP GET method",
		Endpoint:               "",
		Method:                 http.MethodGet,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusOK,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: OkStatusResponse,
	},
	{
		Message:                "Check the wrong entry point to REST API with postfix set to '..'",
		Endpoint:               "..",
		Method:                 http.MethodGet,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusNotFound,
		ExpectedContentType:    ContentTypeText,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the wrong entry point to REST API with postfix set to '../'",
		Endpoint:               "../",
		Method:                 http.MethodGet,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusNotFound,
		ExpectedContentType:    ContentTypeText,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the wrong entry point to REST API with postfix set to '...'",
		Endpoint:               "...",
		Method:                 http.MethodGet,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusNotFound,
		ExpectedContentType:    ContentTypeText,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the wrong entry point to REST API with postfix set to '..?'",
		Endpoint:               "..?",
		Method:                 http.MethodGet,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusNotFound,
		ExpectedContentType:    ContentTypeText,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the non-existent end point in REST API",
		Endpoint:               "foobar",
		Method:                 http.MethodGet,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusNotFound,
		ExpectedContentType:    ContentTypeText,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the entry point to REST API using wrong HTTP method POST",
		Endpoint:               "",
		Method:                 http.MethodPost,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusMethodNotAllowed,
		ExpectedContentType:    None,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the entry point to REST API using wrong HTTP method PUT",
		Endpoint:               "",
		Method:                 http.MethodPut,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusMethodNotAllowed,
		ExpectedContentType:    None,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the entry point to REST API using wrong HTTP method DELETE",
		Endpoint:               "",
		Method:                 http.MethodDelete,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusMethodNotAllowed,
		ExpectedContentType:    None,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the entry point to REST API using wrong HTTP method PATCH",
		Endpoint:               "",
		Method:                 http.MethodPatch,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusMethodNotAllowed,
		ExpectedContentType:    None,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the entry point to REST API using wrong HTTP method HEAD",
		Endpoint:               "",
		Method:                 http.MethodHead,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusMethodNotAllowed,
		ExpectedContentType:    None,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the entry point to REST API using wrong HTTP method OPTIONS",
		Endpoint:               "",
		Method:                 http.MethodOptions,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusMethodNotAllowed,
		ExpectedContentType:    None,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the Prometheus metrics API endpoint",
		Endpoint:               "metrics",
		Method:                 http.MethodGet,
		AuthHeader:             false,
		ExpectedStatus:         http.StatusOK,
		AdditionalChecker:      metricsEndPointContentTypeChecker,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the OpenAPI endpoint",
		Endpoint:               "/openapi.json",
		Method:                 http.MethodGet,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusOK,
		ExpectedContentType:    ContentTypeJSONWithoutCharset,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the info endpoint",
		Endpoint:               "info",
		Method:                 http.MethodGet,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusOK,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: OkStatusResponse,
		AdditionalChecker:      infoResponseChecker,
	},
	{
		Message:                "Check the endpoint to retrieve report for existing organization and cluster ID",
		Endpoint:               constructURLForReportForOrgCluster(knownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusOK,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: OkStatusResponse,
	},
	{
		Message:                "Check the endpoint to retrieve report for existing organization and non-existing cluster ID",
		Endpoint:               constructURLForReportForOrgCluster(knownOrganizationID, unknownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusNotFound,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: "Item with ID 1/00000000-0000-0000-0000-000000000001 was not found in the storage",
	},
	{
		Message:                "Check the endpoint to retrieve report for non-existing organization and existing cluster ID",
		Endpoint:               constructURLForReportForOrgCluster(unknownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             true,
		AuthHeaderOrganization: 100000,
		ExpectedStatus:         http.StatusNotFound,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: "Item with ID 100000/00000000-0000-0000-0000-000000000000 was not found in the storage",
	},
	{
		Message:                "Check the endpoint to retrieve report for non-existing organization and non-existing cluster ID",
		Endpoint:               constructURLForReportForOrgCluster(unknownOrganizationID, unknownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             true,
		AuthHeaderOrganization: 100000,
		ExpectedStatus:         http.StatusNotFound,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: "Item with ID 100000/00000000-0000-0000-0000-000000000001 was not found in the storage",
	},
	{
		Message:                "Reproducer for issue #384 (https://github.com/RedHatInsights/insights-results-aggregator/issues/384)",
		Endpoint:               constructURLForReportForOrgCluster("000000000000000000000000000000000000", "1", testdata.UserID),
		Method:                 http.MethodOptions,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusBadRequest,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: "Error during parsing param 'cluster' with value '1'. Error: 'invalid UUID length: 1'",
	},
	{
		Message:                "Check the endpoint to retrieve report for improper organization",
		Endpoint:               constructURLForReportForOrgCluster(wrongOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusBadRequest,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: "Error during parsing param 'org_id' with value 'foobar'. Error: 'unsigned integer expected'",
	},
	{
		Message:                "Check the endpoint to retrieve report for existing organization and cluster ID w/o authorization token",
		Endpoint:               constructURLForReportForOrgCluster(knownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             false,
		ExpectedStatus:         http.StatusUnauthorized,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: MissingAuthToken,
	},
	{
		Message:                "Check the endpoint to retrieve report for existing organization and non-existing cluster ID w/o authorization token",
		Endpoint:               constructURLForReportForOrgCluster(knownOrganizationID, unknownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             false,
		ExpectedStatus:         http.StatusUnauthorized,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: MissingAuthToken,
	},
	{
		Message:                "Check the endpoint to retrieve report for non-existing organization and existing cluster ID w/o authorization token",
		Endpoint:               constructURLForReportForOrgCluster(unknownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             false,
		ExpectedStatus:         http.StatusUnauthorized,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: MissingAuthToken,
	},
	{
		Message:                "Check the endpoint to retrieve report for non-existing organization and non-existing cluster ID w/o authorization token",
		Endpoint:               constructURLForReportForOrgCluster(unknownOrganizationID, unknownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             false,
		ExpectedStatus:         http.StatusUnauthorized,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: MissingAuthToken,
	},
	{
		Message:                "Check the endpoint to retrieve report for improper organization and known cluster w/o authorization token",
		Endpoint:               constructURLForReportForOrgCluster(wrongOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             false,
		ExpectedStatus:         http.StatusUnauthorized,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: MissingAuthToken,
	},
	{
		Message:                "Check the endpoint to retrieve report for improper organization and unknown cluster w/o authorization token",
		Endpoint:               constructURLForReportForOrgCluster(wrongOrganizationID, unknownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             false,
		ExpectedStatus:         http.StatusUnauthorized,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: MissingAuthToken,
	},
	{
		Message:                "Check the endpoint to retrieve reports using wrong HTTP method POST",
		Endpoint:               constructURLForReportForOrgCluster(knownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodPost,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusMethodNotAllowed,
		ExpectedContentType:    None,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the endpoint to retrieve reports using wrong HTTP method PUT",
		Endpoint:               constructURLForReportForOrgCluster(knownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodPut,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusMethodNotAllowed,
		ExpectedContentType:    None,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the endpoint to retrieve reports using wrong HTTP method DELETE",
		Endpoint:               constructURLForReportForOrgCluster(knownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodDelete,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusMethodNotAllowed,
		ExpectedContentType:    None,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the endpoint to retrieve reports using wrong HTTP method PATCH",
		Endpoint:               constructURLForReportForOrgCluster(knownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodPatch,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusMethodNotAllowed,
		ExpectedContentType:    None,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the endpoint to retrieve reports using wrong HTTP method HEAD",
		Endpoint:               constructURLForReportForOrgCluster(knownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodHead,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusMethodNotAllowed,
		ExpectedContentType:    None,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the endpoint to retrieve reports using correct HTTP method OPTIONS",
		Endpoint:               constructURLForReportForOrgCluster(knownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodOptions,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusOK,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: OkStatusResponse,
	},
	{
		Message:                "Check the endpoint to retrieve report metadata for existing organization and cluster ID",
		Endpoint:               constructURLForReportInfoForOrgCluster(knownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusOK,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: OkStatusResponse,
	},
	{
		Message:                "Check the endpoint to retrieve report metadata for existing organization and non-existing cluster ID",
		Endpoint:               constructURLForReportInfoForOrgCluster(knownOrganizationID, unknownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusNotFound,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: "Item with ID 1/00000000-0000-0000-0000-000000000001 was not found in the storage",
	},
	{
		Message:                "Check the endpoint to retrieve report metadata for unknown organization and cluster ID",
		Endpoint:               constructURLForReportInfoForOrgCluster(unknownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusNotFound,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: "Item with ID 100000/00000000-0000-0000-0000-000000000000 was not found in the storage",
	},
	{
		Message:                "Check the endpoint to retrieve report metadata for unknown organization and non-existing cluster ID",
		Endpoint:               constructURLForReportInfoForOrgCluster(unknownOrganizationID, unknownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusNotFound,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: "Item with ID 100000/00000000-0000-0000-0000-000000000001 was not found in the storage",
	},
	{
		Message:                "Check the endpoint to retrieve report metadata for improper organization and known cluster ID",
		Endpoint:               constructURLForReportInfoForOrgCluster(wrongOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusBadRequest,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: "Error during parsing param 'org_id' with value 'foobar'. Error: 'unsigned integer expected'",
	},
	{
		Message:                "Check the endpoint to retrieve report metadata for improper organization and unknown cluster ID",
		Endpoint:               constructURLForReportInfoForOrgCluster(wrongOrganizationID, unknownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusBadRequest,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: "Error during parsing param 'org_id' with value 'foobar'. Error: 'unsigned integer expected'",
	},
	{
		Message:                "Check the endpoint to retrieve report metadata for existing organization and cluster ID w/o authorization token",
		Endpoint:               constructURLForReportInfoForOrgCluster(knownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             false,
		ExpectedStatus:         http.StatusUnauthorized,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: MissingAuthToken,
	},
	{
		Message:                "Check the endpoint to retrieve report metadata for existing organization and unknown cluster ID w/o authorization token",
		Endpoint:               constructURLForReportInfoForOrgCluster(knownOrganizationID, unknownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             false,
		ExpectedStatus:         http.StatusUnauthorized,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: MissingAuthToken,
	},
	{
		Message:                "Check the endpoint to retrieve report metadata for unknown organization and cluster ID w/o authorization token",
		Endpoint:               constructURLForReportInfoForOrgCluster(unknownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             false,
		ExpectedStatus:         http.StatusUnauthorized,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: MissingAuthToken,
	},
	{
		Message:                "Check the endpoint to retrieve report metadata for unknown organization and unknown cluster ID w/o authorization token",
		Endpoint:               constructURLForReportInfoForOrgCluster(unknownOrganizationID, unknownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             false,
		ExpectedStatus:         http.StatusUnauthorized,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: MissingAuthToken,
	},
	{
		Message:                "Check the endpoint to retrieve report metadata for improper organization and cluster ID w/o authorization token",
		Endpoint:               constructURLForReportInfoForOrgCluster(wrongOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             false,
		ExpectedStatus:         http.StatusUnauthorized,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: MissingAuthToken,
	},
	{
		Message:                "Check the endpoint to retrieve report metadata for improper organization and unknown cluster ID w/o authorization token",
		Endpoint:               constructURLForReportInfoForOrgCluster(wrongOrganizationID, unknownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodGet,
		AuthHeader:             false,
		ExpectedStatus:         http.StatusUnauthorized,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: MissingAuthToken,
	},
	{
		Message:                "Check the endpoint to retrieve report metadata using wrong HTTP method POST",
		Endpoint:               constructURLForReportInfoForOrgCluster(knownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodPost,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusMethodNotAllowed,
		ExpectedContentType:    None,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the endpoint to retrieve report metadata using wrong HTTP method PUT",
		Endpoint:               constructURLForReportInfoForOrgCluster(knownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodPut,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusMethodNotAllowed,
		ExpectedContentType:    None,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the endpoint to retrieve report metadata using wrong HTTP method DELETE",
		Endpoint:               constructURLForReportInfoForOrgCluster(knownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodDelete,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusMethodNotAllowed,
		ExpectedContentType:    None,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the endpoint to retrieve report metadata using wrong HTTP method PATCH",
		Endpoint:               constructURLForReportInfoForOrgCluster(knownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodPatch,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusMethodNotAllowed,
		ExpectedContentType:    None,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the endpoint to retrieve report metadata using wrong HTTP method HEAD",
		Endpoint:               constructURLForReportInfoForOrgCluster(knownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodHead,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusMethodNotAllowed,
		ExpectedContentType:    None,
		ExpectedResponseStatus: None,
	},
	{
		Message:                "Check the endpoint to retrieve report metadata using correct HTTP method OPTIONS",
		Endpoint:               constructURLForReportInfoForOrgCluster(knownOrganizationID, knownClusterForOrganization1, testdata.UserID),
		Method:                 http.MethodOptions,
		AuthHeader:             true,
		ExpectedStatus:         http.StatusOK,
		ExpectedContentType:    ContentTypeJSON,
		ExpectedResponseStatus: OkStatusResponse,
	},
}

func main() {
	os.Exit(runAllTests(tests))
}
