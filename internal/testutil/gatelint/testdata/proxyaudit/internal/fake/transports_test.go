package fake

import "net/http"

// Test files are out of scope: tests build bare transports on purpose.
var testOnly = &http.Transport{}
