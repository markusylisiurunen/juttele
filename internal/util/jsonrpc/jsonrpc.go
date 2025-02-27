package jsonrpc

import "encoding/json"

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
	id     int
	method string
	params any
}

func NewRequest(id int, method string, params any) Request {
	return Request{id, method, params}
}

func (r Request) MarshalJSON() ([]byte, error) {
	type tmp struct {
		Schema string `json:"jsonrpc"`
		Id     int    `json:"id"`
		Method string `json:"method"`
		Params any    `json:"params"`
	}
	var v tmp
	v.Schema = "2.0"
	v.Id = r.id
	v.Method = r.method
	v.Params = r.params
	return json.Marshal(v)
}

type Response struct {
	id           int
	result       any
	errorCode    *int
	errorMessage *string
}

func NewResult(id int, result any) Response {
	return Response{id, result, nil, nil}
}

func NewError(id int, code int, message string) Response {
	return Response{id, nil, &code, &message}
}

func (r Response) MarshalJSON() ([]byte, error) {
	if r.errorCode != nil {
		type tmp struct {
			Schema string `json:"jsonrpc"`
			Id     int    `json:"id"`
			Error  struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		var v tmp
		v.Schema = "2.0"
		v.Id = r.id
		v.Error.Code = *r.errorCode
		v.Error.Message = *r.errorMessage
		return json.Marshal(v)
	}
	type tmp struct {
		Schema string `json:"jsonrpc"`
		Id     int    `json:"id"`
		Result any    `json:"result"`
	}
	var v tmp
	v.Schema = "2.0"
	v.Id = r.id
	v.Result = r.result
	return json.Marshal(v)
}
