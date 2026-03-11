package jsonrpc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewStringID(t *testing.T) {
	id := NewStringID("abc")
	if id.IsNull() {
		t.Fatal("expected non-null")
	}
	if id.Value() != "abc" {
		t.Fatalf("expected abc, got %v", id.Value())
	}
}

func TestNewNumberID(t *testing.T) {
	id := NewNumberID(42)
	if id.IsNull() {
		t.Fatal("expected non-null")
	}
	if id.Value() != int64(42) {
		t.Fatalf("expected 42, got %v", id.Value())
	}
}

func TestNullID(t *testing.T) {
	id := NullID()
	if !id.IsNull() {
		t.Fatal("expected null")
	}
	if id.Value() != nil {
		t.Fatalf("expected nil, got %v", id.Value())
	}
}

func TestIDMarshalJSON_String(t *testing.T) {
	id := NewStringID("test-id")
	data, err := json.Marshal(id)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `"test-id"` {
		t.Fatalf("expected \"test-id\", got %s", data)
	}
}

func TestIDMarshalJSON_Number(t *testing.T) {
	id := NewNumberID(99)
	data, err := json.Marshal(id)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "99" {
		t.Fatalf("expected 99, got %s", data)
	}
}

func TestIDMarshalJSON_Null(t *testing.T) {
	id := NullID()
	data, err := json.Marshal(id)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "null" {
		t.Fatalf("expected null, got %s", data)
	}
}

func TestIDUnmarshalJSON_String(t *testing.T) {
	var id ID
	if err := json.Unmarshal([]byte(`"hello"`), &id); err != nil {
		t.Fatal(err)
	}
	if id.Value() != "hello" {
		t.Fatalf("expected hello, got %v", id.Value())
	}
}

func TestIDUnmarshalJSON_Number(t *testing.T) {
	var id ID
	if err := json.Unmarshal([]byte(`123`), &id); err != nil {
		t.Fatal(err)
	}
	if id.Value() != int64(123) {
		t.Fatalf("expected 123, got %v", id.Value())
	}
}

func TestIDUnmarshalJSON_Null(t *testing.T) {
	var id ID
	if err := json.Unmarshal([]byte(`null`), &id); err != nil {
		t.Fatal(err)
	}
	if !id.IsNull() {
		t.Fatal("expected null")
	}
}

func TestIDUnmarshalJSON_Float(t *testing.T) {
	var id ID
	if err := json.Unmarshal([]byte(`1.5`), &id); err != nil {
		t.Fatal(err)
	}
	if id.Value() != float64(1.5) {
		t.Fatalf("expected 1.5, got %v", id.Value())
	}
}

func TestIDUnmarshalJSON_Invalid(t *testing.T) {
	var id ID
	err := json.Unmarshal([]byte(`[1,2]`), &id)
	if err == nil {
		t.Fatal("expected error for invalid id")
	}
}

func TestRequestMarshal(t *testing.T) {
	id := NewNumberID(1)
	req := Request{
		JSONRPC: Version,
		Method:  "test",
		ID:      &id,
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	json.Unmarshal(data, &parsed)
	if parsed["jsonrpc"] != "2.0" {
		t.Fatalf("expected jsonrpc 2.0, got %v", parsed["jsonrpc"])
	}
	if parsed["method"] != "test" {
		t.Fatalf("expected method test, got %v", parsed["method"])
	}
}

func TestRequestUnmarshal(t *testing.T) {
	raw := `{"jsonrpc":"2.0","method":"foo","params":{"bar":"baz"},"id":1}`
	var req Request
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
	if req.JSONRPC != "2.0" {
		t.Fatalf("expected 2.0, got %s", req.JSONRPC)
	}
	if req.Method != "foo" {
		t.Fatalf("expected foo, got %s", req.Method)
	}
	if req.ID == nil {
		t.Fatal("expected non-nil id")
	}
	if req.ID.Value() != int64(1) {
		t.Fatalf("expected id 1, got %v", req.ID.Value())
	}
	if req.Params == nil {
		t.Fatal("expected params")
	}
}

func TestRequestIsNotification(t *testing.T) {
	req := Request{JSONRPC: Version, Method: "notify"}
	if !req.IsNotification() {
		t.Fatal("expected notification when ID is nil")
	}

	id := NewNumberID(1)
	req.ID = &id
	if req.IsNotification() {
		t.Fatal("expected non-notification when ID is set")
	}
}

func TestResponseMarshal(t *testing.T) {
	id := NewStringID("abc")
	resp := NewResponse(&id, map[string]string{"ok": "true"})
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]json.RawMessage
	json.Unmarshal(data, &parsed)
	if string(parsed["jsonrpc"]) != `"2.0"` {
		t.Fatalf("unexpected jsonrpc: %s", parsed["jsonrpc"])
	}
	if _, ok := parsed["error"]; ok {
		t.Fatal("expected no error field")
	}
}

func TestResponseUnmarshal(t *testing.T) {
	raw := `{"jsonrpc":"2.0","result":{"value":42},"id":"test"}`
	var resp Response
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error != nil {
		t.Fatal("expected no error")
	}
	if resp.ID == nil || resp.ID.Value() != "test" {
		t.Fatalf("unexpected id: %v", resp.ID)
	}
}

func TestErrorType(t *testing.T) {
	e := &Error{Code: MethodNotFound, Message: "not found"}
	s := e.Error()
	if !strings.Contains(s, "-32601") {
		t.Fatalf("expected error code in string, got %s", s)
	}
	if !strings.Contains(s, "not found") {
		t.Fatalf("expected message in string, got %s", s)
	}
}

func TestErrorCodes(t *testing.T) {
	cases := []struct {
		name string
		code int
	}{
		{"ParseError", ParseError},
		{"InvalidRequest", InvalidRequest},
		{"MethodNotFound", MethodNotFound},
		{"InvalidParams", InvalidParams},
		{"InternalError", InternalError},
	}
	for _, tc := range cases {
		if tc.code >= 0 {
			t.Errorf("%s should be negative, got %d", tc.name, tc.code)
		}
	}
}

func TestNewResponse(t *testing.T) {
	id := NewNumberID(5)
	resp := NewResponse(&id, "hello")
	if resp.JSONRPC != Version {
		t.Fatalf("expected version %s, got %s", Version, resp.JSONRPC)
	}
	if resp.Error != nil {
		t.Fatal("expected no error")
	}
	if resp.ID.Value() != int64(5) {
		t.Fatalf("expected id 5, got %v", resp.ID.Value())
	}
}

func TestNewError(t *testing.T) {
	id := NewNumberID(10)
	resp := NewError(&id, InvalidParams, "bad params")
	if resp.Result != nil {
		t.Fatal("expected no result")
	}
	if resp.Error == nil {
		t.Fatal("expected error")
	}
	if resp.Error.Code != InvalidParams {
		t.Fatalf("expected code %d, got %d", InvalidParams, resp.Error.Code)
	}
	if resp.Error.Message != "bad params" {
		t.Fatalf("expected bad params, got %s", resp.Error.Message)
	}
}

func TestNewErrorWithData(t *testing.T) {
	id := NewStringID("err")
	resp := NewErrorWithData(&id, InternalError, "fail", map[string]string{"detail": "something"})
	if resp.Error == nil {
		t.Fatal("expected error")
	}
	if resp.Error.Data == nil {
		t.Fatal("expected error data")
	}
	var data map[string]string
	json.Unmarshal(resp.Error.Data, &data)
	if data["detail"] != "something" {
		t.Fatalf("expected something, got %s", data["detail"])
	}
}

func TestNewError_NilID(t *testing.T) {
	resp := NewError(nil, ParseError, "parse error")
	if resp.ID != nil {
		t.Fatal("expected nil id")
	}
}

func TestNotificationMarshal(t *testing.T) {
	n := Notification{
		JSONRPC: Version,
		Method:  "notifications/initialized",
	}
	data, err := json.Marshal(n)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	json.Unmarshal(data, &parsed)
	if _, hasID := parsed["id"]; hasID {
		t.Fatal("notification should not have id")
	}
}

func TestNotificationUnmarshal(t *testing.T) {
	raw := `{"jsonrpc":"2.0","method":"update","params":{"key":"val"}}`
	var n Notification
	if err := json.Unmarshal([]byte(raw), &n); err != nil {
		t.Fatal(err)
	}
	if n.Method != "update" {
		t.Fatalf("expected update, got %s", n.Method)
	}
}

func TestRequestWithEmptyParams(t *testing.T) {
	raw := `{"jsonrpc":"2.0","method":"ping","id":1}`
	var req Request
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
	if req.Params != nil {
		t.Fatal("expected nil params for missing params field")
	}
}

func TestRequestWithNullParams(t *testing.T) {
	raw := `{"jsonrpc":"2.0","method":"ping","params":null,"id":1}`
	var req Request
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
}

func TestLargePayload(t *testing.T) {
	bigStr := strings.Repeat("x", 100000)
	id := NewNumberID(1)
	resp := NewResponse(&id, bigStr)
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var resp2 Response
	if err := json.Unmarshal(data, &resp2); err != nil {
		t.Fatal(err)
	}
}

func TestIDRoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"string", `"abc"`},
		{"number", `42`},
		{"null", `null`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var id ID
			if err := json.Unmarshal([]byte(tc.input), &id); err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(id)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != tc.input {
				t.Fatalf("roundtrip mismatch: %s != %s", data, tc.input)
			}
		})
	}
}
