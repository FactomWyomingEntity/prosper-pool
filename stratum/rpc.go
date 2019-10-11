package stratum

import (
	"encoding/json"
	"fmt"
	"math/rand"
)

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

// TODO: If we store the json.RawMessage, it would give us a performance boost
func (r Request) FitParams(t interface{}) error {
	data, err := json.Marshal(r.Params)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, t)
}

type SubscribeParams []string

func SubscribeRequest() Request {
	return Request{
		ID:     rand.Int(),
		Method: "mining.subscribe",
		// TODO: We need to compile in the version and come up with a name
		Params: SubscribeParams{"privpool/0.1.0", ""},
	}
}

type Response struct {
	ID     int         `json:"id"`
	Result interface{} `json:"result"`
	Error  *RPCError   `json:"error,omitempty"`
}

// TODO: If we store the json.RawMessage, it would give us a performance boost
func (r Response) FitResult(t interface{}) error {
	data, err := json.Marshal(r.Result)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, t)
}

// SubscribeResult is [session id, extranonce1]
type SubscribeResult []string

func SubscribeResponse(id int, session string, nonce uint32) Response {
	return Response{
		ID: id,
		Result: SubscribeResult{
			session, fmt.Sprintf("%x", nonce),
		},
	}
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

const (
	ErrorParseError     = 32700
	ErrorInvalidRequest = 32600
	ErrorMethodNotFound = 32601
	ErrorInvalidParams  = 32602
	ErrorInternalError  = 32603

	ErrorUnknownException = 1
	ErrorServiceNotFound  = 2
	//ErrorMethodNotFound       = 3
	ErrorFeeRequired          = 10
	ErrorSignatureRequired    = 20
	ErrorSignatureUnavailable = 21
	ErrorUnknownSignatureType = 22
	ErrorBadSignature         = 23
)

func RPCErrorString(errorType int) string {
	switch errorType {
	case ErrorParseError:
		return "ErrorParseError"
	case ErrorInvalidRequest:
		return "ErrorInvalidRequest"
	case ErrorInvalidParams:
		return "ErrorInvalidParams"
	case ErrorInternalError:
		return "ErrorInternalError"
	case ErrorUnknownException:
		return "ErrorUnknownException"
	case ErrorServiceNotFound:
		return "ErrorServiceNotFound"
	case ErrorMethodNotFound:
		return "ErrorMethodNotFound"
	case ErrorFeeRequired:
		return "ErrorFeeRequired"
	case ErrorSignatureRequired:
		return "ErrorSignatureRequired"
	case ErrorSignatureUnavailable:
		return "ErrorSignatureUnavailable"
	case ErrorUnknownSignatureType:
		return "ErrorUnknownSignatureType"
	case ErrorBadSignature:
		return "ErrorBadSignature"
	default:
		return "unknown error"
	}
}

func QuickRPCError(id int, errorType int) Response {
	return Response{
		ID: id,
		Error: &RPCError{
			Code:    errorType,
			Message: RPCErrorString(errorType),
		},
	}
}

// -1, Unknown exception, error message should contain more specific description
// -2, “Service not found”
// -3, “Method not found”
// -10, “Fee required”
// -20, “Signature required”, when server expects request to be signed
// -21, “Signature unavailable”, when server rejects to sign response
// -22, “Unknown signature type”, when server doesn’t understand any signature type from “sign_type”
// -23, “Bad signature”, signature doesn’t match source data
