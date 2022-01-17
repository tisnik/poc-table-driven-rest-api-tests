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
