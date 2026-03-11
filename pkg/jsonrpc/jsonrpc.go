package jsonrpc

import (
	"encoding/json"
	"fmt"
)

const Version = "2.0"

const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

type ID struct {
	value any
}

func NewStringID(s string) ID { return ID{value: s} }
func NewNumberID(n int64) ID  { return ID{value: n} }
func NullID() ID              { return ID{value: nil} }
func (id ID) IsNull() bool    { return id.value == nil }
func (id ID) Value() any      { return id.value }

func (id ID) MarshalJSON() ([]byte, error) {
	if id.value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(id.value)
}

func (id *ID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		id.value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		id.value = s
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		if i, err := n.Int64(); err == nil {
			id.value = i
			return nil
		}
		if f, err := n.Float64(); err == nil {
			id.value = f
			return nil
		}
	}
	return fmt.Errorf("invalid JSON-RPC id: %s", string(data))
}

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      *ID             `json:"id,omitempty"`
}

func (r *Request) IsNotification() bool {
	return r.ID == nil
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
	ID      *ID             `json:"id"`
}

type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("jsonrpc error %d: %s", e.Code, e.Message)
}

type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

func NewResponse(id *ID, result any) *Response {
	data, _ := json.Marshal(result)
	return &Response{
		JSONRPC: Version,
		Result:  data,
		ID:      id,
	}
}

func NewError(id *ID, code int, message string) *Response {
	return &Response{
		JSONRPC: Version,
		Error: &Error{
			Code:    code,
			Message: message,
		},
		ID: id,
	}
}

func NewErrorWithData(id *ID, code int, message string, data any) *Response {
	d, _ := json.Marshal(data)
	return &Response{
		JSONRPC: Version,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    d,
		},
		ID: id,
	}
}
