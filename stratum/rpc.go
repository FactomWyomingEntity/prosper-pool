package stratum

type Request struct {
	ID int `json:"id"`
	Method string `json:"method"`
	Params interface{} `json:"params"`
}

type Response struct {
	ID int `json:"id"`
	Result interface{} `json:"result"`
	Error *RPCError `json:"error"`
}

type RPCError struct {
	Code int `json:"code"`
	Message string `json:"message"`
	Data interface{} `json:"data"`
}

const (
	ErrorUknownException = 1
	ErrorServiceNotFound = 2
	ErrorMethodNotFound = 3
	ErrorFeeRequired = 10
	ErrorSignatureRequired = 20
	ErrorSignatureUnavailable = 21
	ErrorUnknownSignatureType = 22
	ErrorBadSignature = 23
)

// -1, Unknown exception, error message should contain more specific description
// -2, “Service not found”
// -3, “Method not found”
// -10, “Fee required”
// -20, “Signature required”, when server expects request to be signed
// -21, “Signature unavailable”, when server rejects to sign response
// -22, “Unknown signature type”, when server doesn’t understand any signature type from “sign_type”
// -23, “Bad signature”, signature doesn’t match source data