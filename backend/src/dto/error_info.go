package dto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/ztrue/tracerr"
)

type ApiError interface {
	error
}

type ErrorInfo struct {
	Code        ErrorCode       `json:"code"`
	Message     string          `json:"message,omitempty"`
	DeepMessage string          `json:"deep_message,omitempty"`
	Data        map[string]any  `json:"data,omitempty"`
	Err         error           `json:"error,omitempty"`
	Trace       []tracerr.Frame `json:"trace,omitempty"`
}

func NewErrorInfo(code ErrorCode, data map[string]any, err error) *ErrorInfo {
	ret := &ErrorInfo{
		Code: code,
		Data: data,
		Err:  err,
	}
	if err != nil {
		ret.Trace = tracerr.StackTrace(err)
	}
	ret.popolateMessage()
	return ret
}

func (u *ErrorInfo) popolateMessage() error {
	if u.Message == "" {
		tmpl, err := template.New("json").Parse(u.Code.ErrorMessage)
		if err != nil {
			return tracerr.Wrap(err)
		}
		buf := bytes.NewBufferString("")
		err = tmpl.Execute(buf, u.Data)
		if err != nil {
			return tracerr.Wrap(err)
		}
		u.Message = buf.String()
	}
	if u.Err != nil {
		u.DeepMessage = tracerr.Sprint(u.Err)
	}

	return nil
}

func (u ErrorInfo) MarshalJSON() ([]byte, error) {
	err := u.popolateMessage()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	return json.Marshal(u)
}

func (u *ErrorInfo) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, u); err != nil {
		return err
	}
	return nil
}

func (r *ErrorInfo) Error() string {
	err := r.popolateMessage()
	if err != nil {
		return fmt.Sprintf("%x: Internal '%s'", r.Code.errorCode, err.Error())
	}
	return fmt.Sprintf("%x: %s", r.Code.errorCode, r.Message)
}
