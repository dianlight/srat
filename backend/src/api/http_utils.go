package api

import (
	"encoding/json"
	"net/http"

	"github.com/thoas/go-funk"
)

type Options struct {
	Code int
}

type errorResponse struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
	Body  any    `json:"body"`
}

func HttpJSONReponse(w http.ResponseWriter, src any, opt *Options) error {
	if opt == nil {
		opt = &Options{}
	}

	if erx, ok := src.(error); ok {
		opt.Code = funk.GetOrElse(opt.Code, http.StatusInternalServerError).(int)
		return HttpJSONReponse(w, errorResponse{Error: erx.Error(), Body: erx}, opt)
	}

	if src == nil || funk.IsEmpty(src) {
		opt.Code = funk.GetOrElse(opt.Code, http.StatusNoContent).(int)
		w.WriteHeader(opt.Code)
	} else {
		jsonResponse, jsonError := json.Marshal(src)
		if jsonError != nil {
			opt.Code = http.StatusInternalServerError
			return HttpJSONReponse(w, errorResponse{Error: "Unable to encode JSON", Body: jsonError}, opt)
		} else {
			opt.Code = funk.GetOrElse(opt.Code, http.StatusOK).(int)
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
