package stratum

// UnknownRPC is the struct any json rpc can be unmarshalled into before it is catagorized.
type UnknownRPC struct {
	ID int `json:"id"`
	Request
	Response
}

func (u UnknownRPC) GetResponse() Response {
	u.Response.ID = u.ID
	return u.Response
}

func (u UnknownRPC) GetRequest() Request {
	u.Request.ID = u.ID
	return u.Request
}

func (u UnknownRPC) IsRequest() bool {
	return u.Request.Method != ""
}

type Request struct {
	ID     int         `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

type Response struct {
	ID     int         `json:"id"`
	Result interface{} `json:"result"`
	Error  *RPCError   `json:"error"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

const (
	ErrorUnknownException     = 1
	ErrorServiceNotFound      = 2
	ErrorMethodNotFound       = 3
	ErrorFeeRequired          = 10
	ErrorSignatureRequired    = 20
	ErrorSignatureUnavailable = 21
	ErrorUnknownSignatureType = 22
	ErrorBadSignature         = 23
)

// -1, Unknown exception, error message should contain more specific description
// -2, “Service not found”
// -3, “Method not found”
// -10, “Fee required”
// -20, “Signature required”, when server expects request to be signed
// -21, “Signature unavailable”, when server rejects to sign response
// -22, “Unknown signature type”, when server doesn’t understand any signature type from “sign_type”
// -23, “Bad signature”, signature doesn’t match source data
