package stratum

import (
	"encoding/json"
	"math/rand"
)

// UnknownRPC is the struct any json rpc can be unmarshalled into before it is categorized.
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

type RPCParams []string

// Client-to-server methods

func AuthorizeRequest(username, password string) Request {
	return Request{
		ID:     rand.Int(),
		Method: "mining.authorize",
	}.SetParams(RPCParams{username, password})
}

func GetOPRHashRequest(jobID string) Request {
	return Request{
		ID:     rand.Int(),
		Method: "mining.get_oprhash",
	}.SetParams(RPCParams{jobID})
}

func SubmitRequest(username, jobID, nonce, oprHash string) Request {
	return Request{
		ID:     rand.Int(),
		Method: "mining.submit",
	}.SetParams(RPCParams{username, jobID, nonce, oprHash})
}

func SubscribeRequest() Request {
	return Request{
		ID:     rand.Int(),
		Method: "mining.subscribe",
	}.SetParams(RPCParams{"prosper/0.1.0"})
}

func SuggestDifficultyRequest(preferredDifficulty string) Request {
	return Request{
		ID:     rand.Int(),
		Method: "mining.suggest_difficulty",
	}.SetParams(RPCParams{preferredDifficulty})
}

// Server-to-client methods

func GetVersionRequest() Request {
	return Request{
		ID:     rand.Int(),
		Method: "client.get_version",
	}.SetParams(nil)
}

func ReconnectRequest(hostname, port, waittime string) Request {
	return Request{
		ID:     rand.Int(),
		Method: "client.reconnect",
	}.SetParams(RPCParams{hostname, port, waittime})
}

func ShowMessageRequest(message string) Request {
	return Request{
		ID:     rand.Int(),
		Method: "client.show_message",
	}.SetParams(RPCParams{message})
}

func NotifyRequest(jobID, oprHash, cleanjobs string) Request {
	return Request{
		ID:     rand.Int(),
		Method: "mining.notify",
	}.SetParams(RPCParams{jobID, oprHash, cleanjobs})
}

func SetDifficultyRequest(difficulty string) Request {
	return Request{
		ID:     rand.Int(),
		Method: "mining.set_difficulty",
	}.SetParams(RPCParams{difficulty})
}

func SetNonceRequest(nonce string) Request {
	return Request{
		ID:     rand.Int(),
		Method: "mining.set_nonce",
	}.SetParams(RPCParams{nonce})
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

type Subscription struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

// SubscribeResult is [session id, nonce]
type SubscribeResult []Subscription

func AuthorizeResponse(id int, result bool, err error) Response {
	return Response{
		ID: id,
	}.SetResult(result)
}

func SubmitResponse(id int, result bool, err error) Response {
	return Response{
		ID: id,
	}.SetResult(result)
}

func SubscribeResponse(id int, session string) Response {
	notifySub := Subscription{Id: session, Type: "mining.notify"}
	setDiffSub := Subscription{Id: session, Type: "mining.set_difficulty"}

	res := make([]Subscription, 2)
	res[0] = notifySub
	res[1] = setDiffSub
	return Response{
		ID: id,
	}.SetResult(res)
}

func GetVersionResponse(id int, version string) Response {
	return Response{
		ID: id,
	}.SetResult(version)
}

func GetOPRHashResponse(id int, oprHash string) Response {
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
