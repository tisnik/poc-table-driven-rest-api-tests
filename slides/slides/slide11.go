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
    ...
    ...
    ...
    {
        Message:                "Check the endpoint to retrieve report for improper organization",
        Endpoint:               constructURLForReportForOrgCluster(wrongOrganizationID, knownClusterForOrganization1, testdata.UserID),
        Method:                 http.MethodGet,
        AuthHeader:             true,
        ExpectedStatus:         http.StatusBadRequest,
        ExpectedContentType:    ContentTypeJSON,
        ExpectedResponseStatus: "Error during parsing param 'org_id' with value 'foobar'. Error: 'unsigned integer expected'",
    },
