package main

import (
	"default-ndots-admission-controller/internal"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strconv"
	"time"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)
}

func main() {
	certFile := os.Getenv("TLS_CERT")
	keyFile := os.Getenv("TLS_KEY")
	if certFile == "" || keyFile == "" {
		log.Error("TLS_CERT and TLS_KEY environment variables are required.")
		return
	}
	var (
		requestTimeoutDuration time.Duration
		err                    error
	)
	requestTimeoutEnv := os.Getenv("REQUEST_TIMEOUT")
	if requestTimeoutEnv == "" {
		// Allowed timeout is between 10 - 30s
		// https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#timeouts
		requestTimeoutDuration = time.Second * 20
	} else {
		requestTimeoutDuration, err = time.ParseDuration(requestTimeoutEnv)
		if err != nil {
			log.Error("Unable to parse REQUEST_TIMEOUT. Reason:", err)
			return
		}
	}
	tlsFilesExists := true
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		tlsFilesExists = false
	}
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		tlsFilesExists = false
	}
	if !tlsFilesExists {
		log.Error("TLS_CERT and/or TLS_KEY file(s) not found.")
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		// We listen on port 8443 such that we do not need root privileges or extra capabilities for this server.
		// The Service object will take care of mapping this port to the HTTPS port 443.
		port = "8443"
	}

	ndotsValue := 2
	ndotsValEnv := os.Getenv("NDOTS_VALUE")
	if ndotsValEnv != "" {
		ndotsValue, err = strconv.Atoi(ndotsValEnv)
		if err != nil {
			log.Error(errors.Wrapf(err, "Unable convert NDOTS_VALUE %s to int", ndotsValEnv))
			return
		}
	}
	r := internal.NewHandlers(ndotsValue, requestTimeoutDuration)
	addr := fmt.Sprintf("0.0.0.0:%s", port)
	log.Infof("Default ndots controller starting with the following info - address: %s, request timeout: %s, ndots value: %d", addr, requestTimeoutDuration, ndotsValue)
	if err = http.ListenAndServeTLS(addr, certFile, keyFile, r); err != nil {
		log.Error(err)
	}
}
