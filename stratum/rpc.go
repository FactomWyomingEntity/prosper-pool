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
	ID     int             `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func (r Request) SetParams(p interface{}) Request {
	data, _ := json.Marshal(p)
	r.Params = json.RawMessage(data)
	return r
}

// TODO: If we store the json.RawMessage, it would give us a performance boost
func (r Request) FitParams(t interface{}) error {
	return json.Unmarshal(r.Params, t)
}

type SubscribeParams []string

func SubscribeRequest() Request {
	return Request{
		ID:     rand.Int(),
		Method: "mining.subscribe",
		// TODO: We need to compile in the version and come up with a name
		// Params: SubscribeParams{"privpool/0.1.0", ""},
	}.SetParams(SubscribeParams{"privpool/0.1.0", ""})
}

type AuthorizeParams []string

func AuthorizeRequest(username, password string) Request {
	return Request{
		ID:     rand.Int(),
		Method: "mining.authorize",
		//Params: AuthorizeParams{username, password},
	}.SetParams(AuthorizeParams{username, password})
}

type Response struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *RPCError       `json:"error,omitempty"`
}

func (r Response) SetResult(o interface{}) Response {
	data, _ := json.Marshal(o)
	r.Result = json.RawMessage(data)
	return r
}

func (r Response) FitResult(t interface{}) error {
	return json.Unmarshal(r.Result, t)
}

// SubscribeResult is [session id, extranonce1]
type SubscribeResult []string

func SubscribeResponse(id int, session string, nonce uint32) Response {
	return Response{
		ID: id,
	}.SetResult(SubscribeResult{
		session, fmt.Sprintf("%x", nonce),
	})
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
