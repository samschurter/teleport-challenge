package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net/http"
)

func main() {

	caFile := "../../certs/ca.crt"
	certFile := "../../certs/client.crt"
	keyFile := "../../certs/client.key"

	clientCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("failed to load key pair: %v", err)
	}

	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		log.Fatalf("failed to read CA cert file: %v", err)
	}

	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(caCert); !ok {
		log.Fatalf("failed to create cert pool: %v", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      pool,
				Certificates: []tls.Certificate{clientCert},
			},
		},
	}

	rootCmd.AddCommand(start(client))
	rootCmd.AddCommand(stop(client))
	rootCmd.AddCommand(status(client))
	rootCmd.AddCommand(output(client))
	rootCmd.Execute()
}
