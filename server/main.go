package main

import "flag"

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"golang.org/x/crypto/acme/autocert"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/pikans/mealplan/moira"
)

var deprecatedRSAIncEmailAddressForUseInSignatures = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 1}

func getMITCertEmailAddressFullName(chains [][]*x509.Certificate) (moira.Email, string, error) {
	if len(chains) == 0 {
		return "", "", errors.New("a client certificate is required to use this service, but no verified certificate chains")
	}
	for _, chain := range chains {
		if len(chain) == 0 {
			continue
		}
		cert := chain[0] // leaf
		for _, name := range cert.Subject.Names {
			if !name.Type.Equal(deprecatedRSAIncEmailAddressForUseInSignatures) {
				continue
			}
			if email, ok := name.Value.(string); ok {
				return moira.Email(email), cert.Subject.CommonName, nil
			}
		}
	}
	return "", "", errors.New("no MIT certificate email address found")
}

func run(handler http.Handler, unauthHandler http.Handler, register, listenhttp, listenhttps, authenticate, authorize, state string) {
	letsEncryptManager := &autocert.Manager{
		Cache:      autocert.DirCache(state),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: func(ctx context.Context, host string) error { return nil },
		Email:      register,
	}

	clientCAsPEM, err := ioutil.ReadFile(authenticate)
	if err != nil {
		log.Fatalf("error reading client CAs file: %s", err)
	}
	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(clientCAsPEM) {
		log.Fatalf("failed to parse client CA certificate")
	}

	doAuthorize := func(req *http.Request) error {
		email, fullname, err := getMITCertEmailAddressFullName(req.TLS.VerifiedChains)
		if err != nil {
			return err
		}
		if err := moira.IsAuthorized(authorize, moira.UsernameFromEmail(email)); err != nil {
			return err
		}
		req.Header.Set("proxy-authorized-list", authorize)
		req.Header.Set("proxy-authenticated-full-name", fullname)
		req.Header.Set("proxy-authenticated-email", strings.ToLower(string(email)))
		return nil
	}

	go func() { log.Fatal(http.ListenAndServe(listenhttp, letsEncryptManager.HTTPHandler(nil))) }()
	srv := &http.Server{
		Addr: listenhttps,
		TLSConfig: &tls.Config{
			GetCertificate: letsEncryptManager.GetCertificate,

			ClientCAs:  clientCAs,
			ClientAuth: tls.VerifyClientCertIfGiven,
		},
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if err := doAuthorize(req); err == nil {
				handler.ServeHTTP(w, req)
			} else {
				unauthHandler.ServeHTTP(w, req)
			}
		}),
	}

	log.Fatal(srv.ListenAndServeTLS("", ""))
}

var register = flag.String("register", "", "(optional) email address for letsencrypt registration")
var listenhttp = flag.String("listenhttp", ":http", "host:port to listen for HTTP on")
var listenhttps = flag.String("listenhttps", ":https", "host:port to listen for HTTPS on")
var authenticate = flag.String("authenticate", "", "path to a file containing PEM-format x509 certificates for the CAs trusted to authenticate clients")
var authorize = flag.String("authorize", "", "name of moira list whose members are authorized. The list MUST be marked as a NFS group (blanche listname -N)")
var state = flag.String("state", "", "path at which the letsencrypt server state will be recorded")

func main() {
	flag.Parse()
	if *authenticate == "" || *authorize == "" || *state == "" {
		flag.Usage()
		log.Fatal("please specify the required arguments")
	}
	run(getHandler(), getUnauthHandler(), *register, *listenhttp, *listenhttps, *authenticate, *authorize, *state)
}
