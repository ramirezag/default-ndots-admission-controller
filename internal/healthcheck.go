package internal

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func health(w http.ResponseWriter, _ *http.Request) {
	jsonResponse(w, 200, map[string]string{"health": "UP"})
}

func jsonResponse(w http.ResponseWriter, statusCode int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
	} else {
		w.WriteHeader(statusCode)
		_, err := w.Write(jsonBytes)
		if err != nil {
			log.Error(fmt.Sprintf("Error encountered when writing %v to response, %v", response, err))
		}
	}
}
