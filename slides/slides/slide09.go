

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
