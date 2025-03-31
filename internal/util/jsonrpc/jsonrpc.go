package jsonrpc

import (
	"encoding/json"
	"fmt"
)

type Notification struct {
	method string
	params any
}

func NewNotification(method string, params any) Notification {
	return Notification{method, params}
}

func (n Notification) MarshalJSON() ([]byte, error) {
	type tmp struct {
		Schema string `json:"jsonrpc"`
		Method string `json:"method"`
		Params any    `json:"params"`
	}
	var v tmp
	v.Schema = "2.0"
	v.Method = n.method
	v.Params = n.params
	return json.Marshal(v)
}

type Request struct {
	ID     uint64
	Method string
	Params any
}

func NewRequest(id uint64, method string, params any) Request {
	return Request{id, method, params}
}

func (r Request) MarshalJSON() ([]byte, error) {
	type tmp struct {
		Schema string `json:"jsonrpc"`
		Id     uint64 `json:"id"`
		Method string `json:"method"`
		Params any    `json:"params"`
	}
	var v tmp
	v.Schema = "2.0"
	v.Id = r.ID
	v.Method = r.Method
	v.Params = r.Params
	return json.Marshal(v)
}

func (r *Request) UnmarshalJSON(data []byte) error {
	type tmp struct {
		Schema string `json:"jsonrpc"`
		Id     uint64 `json:"id"`
		Method string `json:"method"`
		Params any    `json:"params"`
	}
	var v tmp
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	if v.Schema != "2.0" {
		return fmt.Errorf("invalid schema: %s", v.Schema)
	}
	if v.Method == "" {
		return fmt.Errorf("invalid method: %s", v.Method)
	}
	r.ID = v.Id
	r.Method = v.Method
	r.Params = v.Params
	return nil
}

type Response struct {
	ID           uint64
	Result       *any
	ErrorCode    *int64
	ErrorMessage *string
}

func NewResult(id uint64, result any) Response {
	return Response{id, &result, nil, nil}
}

func NewError(id uint64, code int64, message string) Response {
	return Response{id, nil, &code, &message}
}

func (r Response) MarshalJSON() ([]byte, error) {
	isResult := r.Result != nil
	isError := r.ErrorCode != nil && r.ErrorMessage != nil
	if !isResult && !isError {
		return nil, fmt.Errorf("invalid response")
	}
	if r.ErrorCode != nil && r.ErrorMessage != nil {
		type tmp struct {
			Schema string `json:"jsonrpc"`
			Id     uint64 `json:"id"`
			Error  struct {
				Code    int64  `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		var v tmp
		v.Schema = "2.0"
		v.Id = r.ID
		v.Error.Code = *r.ErrorCode
		v.Error.Message = *r.ErrorMessage
		return json.Marshal(v)
	}
	type tmp struct {
		Schema string `json:"jsonrpc"`
		Id     uint64 `json:"id"`
		Result any    `json:"result"`
	}
	var v tmp
	v.Schema = "2.0"
	v.Id = r.ID
	v.Result = *r.Result
	return json.Marshal(v)
}

func (r *Response) UnmarshalJSON(data []byte) error {
	type tmp struct {
		Schema string          `json:"jsonrpc"`
		Id     uint64          `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int64  `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	var v tmp
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	if v.Schema != "2.0" {
		return fmt.Errorf("invalid schema: %s", v.Schema)
	}
	isResult := v.Result != nil
	isError := v.Error != nil
	if !isResult && !isError {
		return fmt.Errorf("invalid response")
	}
	r.ID = v.Id
	if v.Error != nil {
		r.ErrorCode = &v.Error.Code
		r.ErrorMessage = &v.Error.Message
	} else {
		var result any = v.Result
		r.Result = &result
	}
	return nil
}
