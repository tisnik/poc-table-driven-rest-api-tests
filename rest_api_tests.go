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

import "fmt"

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
