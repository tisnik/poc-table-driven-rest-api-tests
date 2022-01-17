

// see the commented line
func checkRestAPIEntryPoint() {
	f := frisby.Create("Check the entry point to REST API using HTTP GET method").Get(apiURL)
	setAuthHeader(f)
	f.Send()
	f.ExpectStatus(200)
	// omited f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	f.PrintReport()
}
