package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/dianlight/srat/dto"
	"github.com/thoas/go-funk"
	"github.com/ztrue/tracerr"
)

type Options struct {
	Code int
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

	if erx, ok := src.(dto.ErrorInfo); ok {
		opt.Code = codeGetOrElse(opt.Code, http.StatusInternalServerError)
		slog.Error(erx.Error())
		return HttpJSONReponse(w, erx, opt)
	} else if erx, ok := src.(error); ok {
		opt.Code = codeGetOrElse(opt.Code, http.StatusInternalServerError)
		slog.Error(tracerr.SprintSourceColor(erx))
		return HttpJSONReponse(w, dto.NewErrorInfo(dto.ErrorCodes.GENERIC_ERROR, nil, erx), opt)
	}

	if src == nil || funk.IsEmpty(src) {
		opt.Code = codeGetOrElse(opt.Code, http.StatusNoContent)
		w.WriteHeader(opt.Code)
	} else {
		jsonResponse, jsonError := json.Marshal(src)
		if jsonError != nil {
			opt.Code = http.StatusInternalServerError
			return HttpJSONReponse(w,
				tracerr.Wrap(dto.NewErrorInfo(dto.ErrorCodes.GENERIC_ERROR, nil, jsonError)), opt)
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
		return tracerr.Wrap(err)
	}
	return nil
}
