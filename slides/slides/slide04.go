

// more complicated test
func checkReportEndpointForImproperOrganization() {
	url := constructURLForReportForOrgCluster(wrongOrganizationID,
	                                          knownClusterForOrganization1,
						  testdata.UserID)
	f := frisby.Create("Check the endpoint to return report for improper organization").Get(url)
	setAuthHeader(f)
	f.Send()
	f.ExpectStatus(400)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	// actually this part is refactored into own function
	text, err := f.Resp.Content()
	if err != nil {
		f.AddError(err.Error())
	} else {
		err := json.Unmarshal(text, &response)
		if err != nil {
			f.AddError(err.Error())
		}
	}
	if response.Status == server.OkStatusPayload {
		f.AddError(fmt.Sprintf("Expected error status, but got '%s' instead",
		                       response.Status))
	}

	f.PrintReport()
}
