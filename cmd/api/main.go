package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	gcontext "github.com/gorilla/context"
	"github.com/gorilla/mux"

	"github.com/samschurter/teleport-challenge/pkg/alps"
)

func main() {

	caFile := flag.String("ca", "certs/ca.crt", "")
	certFile := flag.String("crt", "certs/server.crt", "")
	keyFile := flag.String("key", "certs/server.key", "")
	flag.Parse()

	r := mux.NewRouter()

	js := jobServer{
		hub: alps.NewHub(),
	}

	r.HandleFunc("/start", js.start).Methods(http.MethodPost)
	r.HandleFunc("/stop/{id}", js.stop).Methods(http.MethodPost)
	r.HandleFunc("/status/{id}", js.status).Methods(http.MethodGet)
	r.HandleFunc("/stdout/{id}", js.stdout).Methods(http.MethodGet)
	r.HandleFunc("/stderr/{id}", js.stderr).Methods(http.MethodGet)

	r.Use(authorizationMiddleware)
	tlsConf, err := makeConfig(*caFile, *certFile, *keyFile)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Handler:           r,
		Addr:              "localhost:4430",
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
		TLSConfig:         tlsConf,
	}

	log.Fatal(srv.ListenAndServeTLS(*certFile, *keyFile))
}

func authorizationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		certs := r.TLS.PeerCertificates
		if len(certs) == 0 {
			httpError(w, "unauthorized: no certs found", http.StatusUnauthorized)
			return
		}
		c := certs[0]
		orgs := c.Subject.Organization
		if len(orgs) == 0 {
			httpError(w, "unauthorized: no org found", http.StatusUnauthorized)
			return
		}
		org := orgs[0]

		if !authorized(r.URL.Path, org) {
			httpError(w, "unauthorized: you are not allowed to access this path", http.StatusUnauthorized)
			return
		}

		gcontext.Set(r, "user", org)
		next.ServeHTTP(w, r)
	})
}

func authorized(path, org string) bool {

	acl := map[string][]string{
		`\/start`: {
			"samschurter@makeict.org",
		},
		`\/status/.*`: {
			"samschurter@makeict.org",
		},
		`\/stop\/.*`: {
			"samschurter@makeict.org",
		},
		`\/stdout\/.*`: {
			"samschurter@makeict.org",
		},
		`\/stderr\/.*`: {
			"samschurter@makeict.org",
		},
	}

	var users []string
	for p, u := range acl {
		matched, err := regexp.MatchString(p, path)
		if err != nil {
			log.Printf("failed to match string: %v", err)
		}
		if matched {
			users = u
			break
		}
	}
	if len(users) == 0 {
		return false
	}

	for _, u := range users {
		if u == org {
			return true
		}
	}

	return false
}

func makeConfig(caFile, certFile, keyFile string) (*tls.Config, error) {
	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		dir, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(dir)
		return nil, err
	}
	clientCA := x509.NewCertPool()
	if ok := clientCA.AppendCertsFromPEM(caCert); !ok {
		return nil, fmt.Errorf("failed to append certs")
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates:             []tls.Certificate{cert},
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                clientCA,
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
	}, nil
}
