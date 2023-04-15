package main

import (
	"crypto/tls"
	"default-ndots-admission-controller/internal"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"net/http"
	"os"
	"strconv"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)
}

func main() {
	// Start the tracer
	tracer.Start()
	defer tracer.Stop()

	certFile := os.Getenv("TLS_CERT")
	keyFile := os.Getenv("TLS_KEY")
	if certFile == "" || keyFile == "" {
		log.Error("TLS_CERT and TLS_KEY environment variables are required.")
		return
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
	sCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Error(errors.Wrap(err, "Unable to load TLS key pair."))
		return
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{sCert},
		// TODO: uses mutual tls after we agree on what cert the apiserver should use.
		// ClientAuth:   tls.RequireAndVerifyClientCert,
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
	r := internal.NewHandlers(ndotsValue)
	server := &http.Server{
		Addr:      fmt.Sprintf(":%s", port),
		TLSConfig: tlsConfig,
		Handler:   r,
	}

	if err = server.ListenAndServeTLS("", ""); err != nil {
		log.Error(err)
	}
}
