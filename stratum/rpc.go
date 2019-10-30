package stratum

import (
	"encoding/json"
	"fmt"
	"math/rand"
)

// UnknownRPC is the struct any json rpc can be unmarshalled into before it is categorized.
type UnknownRPC struct {
	ID int32 `json:"id"`
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
	ID     int32           `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func (r Request) SetParams(p interface{}) Request {
	data, _ := json.Marshal(p)
	r.Params = json.RawMessage(data)
	return r
}

func (r Request) FitParams(t interface{}) error {
	return json.Unmarshal(r.Params, t)
}

type RPCParams []string

// Client-to-server methods

func AuthorizeRequest(username, password, invitecode, payoutaddress string) Request {
	return Request{
		ID:     rand.Int31(),
		Method: "mining.authorize",
	}.SetParams(RPCParams{username, password, invitecode, payoutaddress})
}

func GetOPRHashRequest(jobID string) Request {
	return Request{
		ID:     rand.Int31(),
		Method: "mining.get_oprhash",
	}.SetParams(RPCParams{jobID})
}

func SubmitRequest(username, jobID, nonce, oprHash, target string) Request {
	return Request{
		ID:     rand.Int31(),
		Method: "mining.submit",
	}.SetParams(RPCParams{username, jobID, nonce, oprHash, target})
}

func SubscribeRequest(version string) Request {
	return Request{
		ID:     rand.Int31(),
		Method: "mining.subscribe",
	}.SetParams(RPCParams{"prosper/" + version})
}

func SuggestTargetRequest(preferredTarget string) Request {
	return Request{
		ID:     rand.Int31(),
		Method: "mining.suggest_target",
	}.SetParams(RPCParams{preferredTarget})
}

// Server-to-client methods

func GetVersionRequest() Request {
	return Request{
		ID:     rand.Int31(),
		Method: "client.get_version",
	}.SetParams(nil)
}

func ReconnectRequest(hostname, port, waittime string) Request {
	return Request{
		ID:     rand.Int31(),
		Method: "client.reconnect",
	}.SetParams(RPCParams{hostname, port, waittime})
}

func ShowMessageRequest(message string) Request {
	return Request{
		ID:     rand.Int31(),
		Method: "client.show_message",
	}.SetParams(RPCParams{message})
}

func NotifyRequest(jobID, oprHash, cleanjobs string) Request {
	return Request{
		ID:     rand.Int31(),
		Method: "mining.notify",
	}.SetParams(RPCParams{jobID, oprHash, cleanjobs})
}

func SetTargetRequest(target string) Request {
	return Request{
		ID:     rand.Int31(),
		Method: "mining.set_target",
	}.SetParams(RPCParams{target})
}

func SetNonceRequest(nonce string) Request {
	return Request{
		ID:     rand.Int31(),
		Method: "mining.set_nonce",
	}.SetParams(RPCParams{nonce})
}

func StopMiningRequest() Request {
	return Request{
		ID:     rand.Int31(),
		Method: "mining.stop_mining",
	}.SetParams(nil)
}

type Response struct {
	ID     int32           `json:"id"`
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

type Subscription struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

// SubscribeResult is [session id, nonce]
type SubscribeResult []Subscription

func AuthorizeResponse(id int32, result bool, err error) Response {
	return Response{
		ID: id,
	}.SetResult(result)
}

func SubmitResponse(id int32, result bool, err error) Response {
	return Response{
		ID: id,
	}.SetResult(result)
}

func SubscribeResponse(id int32, session string, nonce uint32) Response {
	notifySub := Subscription{Id: session, Type: "mining.notify"}
	setTargetSub := Subscription{Id: session, Type: "mining.set_target"}
	setNonce := Subscription{Id: fmt.Sprintf("%d", nonce), Type: "mining.set_nonce"}

	res := make([]Subscription, 3)
	res[0] = notifySub
	res[1] = setTargetSub
	res[2] = setNonce
	return Response{
		ID: id,
	}.SetResult(res)
}

func GetVersionResponse(id int32, version string) Response {
	return Response{
		ID: id,
	}.SetResult(version)
}

func GetOPRHashResponse(id int32, oprHash string) Response {
	return Response{
		ID: id,
	}.SetResult(oprHash)
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

func QuickRPCError(id int32, errorType int) Response {
	return Response{
		ID: id,
		Error: &RPCError{
			Code:    errorType,
			Message: RPCErrorString(errorType),
		},
	}
}

func HelpfulRPCError(id int32, errorType int, data interface{}) Response {
	return Response{
		ID: id,
		Error: &RPCError{
			Code:    errorType,
			Message: RPCErrorString(errorType),
			Data:    data,
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
