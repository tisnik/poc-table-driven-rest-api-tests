## Proof of concept: table-driven REST API tests in Go

### The problem

* REST API tests written in Go and based on Frisby are now standard way in Go ecosystem.
* OTOH tests are usually specified as plain Go sources
    - repetitive patterns
    - it's too easy to omit some checks

### Existing tests

```go
func checkRestAPIEntryPoint() {
	f := frisby.Create("Check the entry point to REST API using HTTP GET method").Get(apiURL)
	setAuthHeader(f)
	f.Send()
	f.ExpectStatus(200)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	f.PrintReport()
}
```

```go
func checkReportEndpointForImproperOrganization() {
	url := constructURLForReportForOrgCluster(wrongOrganizationID, knownClusterForOrganization1, testdata.UserID)
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
		f.AddError(fmt.Sprintf("Expected error status, but got '%s' instead", response.Status))
	}

	f.PrintReport()
}
```


### Check omit by mistake

```go
func checkRestAPIEntryPoint() {
	f := frisby.Create("Check the entry point to REST API using HTTP GET method").Get(apiURL)
	setAuthHeader(f)
	f.Send()
	f.ExpectStatus(200)
	// omited f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	f.PrintReport()
}
```

### Proposed solution

* Table-driven approach
* Use Go arrays/slices of structs to define tests
* Pros
    - syntax checks
    - readability
    - ability to specify callback functions/handlers to be called to perform action or check something
* Cons
    - not much space to reinvent the wheel :)

### Test structure

```go
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
```

* Most items can be left unspecified - default options supported

```go
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
```

### One unified test implementation

```go
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
```

### Test runner is trivial

```go
func runAllTests(tests []RestAPITest) int {
	for _, test := range tests {
		checkEndPoint(&test)
	}
	frisby.Global.PrintReport()
	return frisby.Global.NumErrored
}
```

### Conclusion

* We don't have to reinvent the wheel (= make yet another DSL)
* Still the test specification is perfectly readable and checkable
* Room for improvements
