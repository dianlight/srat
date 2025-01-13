package api

import (
	"encoding/json"
	"net/http"

	"github.com/thoas/go-funk"
)

type Options struct {
	Code int
}

type ErrorResponse struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
	Body  any    `json:"body"`
}

func codeGetOrElse(code int, def int) int {
	if code <= 0 {
		return def
	}
	return code
}

func HttpJSONReponse(w http.ResponseWriter, src any, opt *Options) error {
	if opt == nil {
		opt = &Options{
			Code: -1,
		}
	}

	if erx, ok := src.(error); ok {
		opt.Code = codeGetOrElse(opt.Code, http.StatusInternalServerError)
		return HttpJSONReponse(w, ErrorResponse{Error: erx.Error(), Body: erx}, opt)
	}

	if src == nil || funk.IsEmpty(src) {
		opt.Code = codeGetOrElse(opt.Code, http.StatusNoContent)
		w.WriteHeader(opt.Code)
	} else {
		jsonResponse, jsonError := json.Marshal(src)
		if jsonError != nil {
			opt.Code = http.StatusInternalServerError
			return HttpJSONReponse(w, ErrorResponse{Error: "Unable to encode JSON", Body: jsonError}, opt)
		} else {
			opt.Code = codeGetOrElse(opt.Code, http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(opt.Code)
			w.Write(jsonResponse)
		}
	}
	return nil
}

func HttpJSONRequest(dst any, w http.ResponseWriter, r *http.Request) error {
	err := json.NewDecoder(r.Body).Decode(&dst)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	return nil
}
