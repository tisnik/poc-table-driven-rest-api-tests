

func runAllTests(tests []RestAPITest) int {
        for _, test := range tests {
                checkEndPoint(&test)
        }
        frisby.Global.PrintReport()
        return frisby.Global.NumErrored
}
