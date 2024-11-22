package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"dario.cat/mergo"
	"github.com/gorilla/mux"
)

// The function "listShares" returns a list of shares in JSON format over HTTP.
func listShares(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	jsonResponse, jsonError := json.Marshal(config.Shares)

	if jsonError != nil {
		fmt.Println("Unable to encode JSON")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(jsonError.Error()))
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}

}

func getShare(w http.ResponseWriter, r *http.Request) {
	share := mux.Vars(r)["share_name"]
	w.Header().Set("Content-Type", "application/json")

	data, ok := config.Shares[share]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
	} else {
		jsonResponse, jsonError := json.Marshal(data)

		if jsonError != nil {
			fmt.Println("Unable to encode JSON")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(jsonError.Error()))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(jsonResponse)
		}

	}

}

func createShare(w http.ResponseWriter, r *http.Request) {
	share := mux.Vars(r)["share_name"]
	w.Header().Set("Content-Type", "application/json")

	data, ok := config.Shares[share]
	if ok {
		w.WriteHeader(http.StatusConflict)
		jsonResponse, jsonError := json.Marshal(ResponseError{Error: "Share already exists", Body: data})

		if jsonError != nil {
			fmt.Println("Unable to encode JSON")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(jsonError.Error()))
		} else {
			w.Write(jsonResponse)
		}
	} else {
		var share Share

		err := json.NewDecoder(r.Body).Decode(&share)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// TODO: Create Share

		jsonResponse, jsonError := json.Marshal(share)

		if jsonError != nil {
			fmt.Println("Unable to encode JSON")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(jsonError.Error()))
		} else {
			w.WriteHeader(http.StatusCreated)
			w.Write(jsonResponse)
		}

	}
}

func updateShare(w http.ResponseWriter, r *http.Request) {
	share := mux.Vars(r)["share_name"]
	w.Header().Set("Content-Type", "application/json")

	data, ok := config.Shares[share]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
	} else {
		var share Share

		err := json.NewDecoder(r.Body).Decode(&share)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		mergo.Merge(&share, data)

		// TODO: Save share as new data!

		jsonResponse, jsonError := json.Marshal(share)

		if jsonError != nil {
			fmt.Println("Unable to encode JSON")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(jsonError.Error()))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(jsonResponse)
		}

	}

}

func deleteShare(w http.ResponseWriter, r *http.Request) {
	share := mux.Vars(r)["share_name"]
	w.Header().Set("Content-Type", "application/json")

	_, ok := config.Shares[share]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
	} else {

		// TODO: Delete share

		w.WriteHeader(http.StatusNoContent)

	}

}
