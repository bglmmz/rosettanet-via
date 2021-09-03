package creds

import (
	tls "github.com/bglmmz/gmsm/gmtls"
	"github.com/bglmmz/gmsm/x509"
	"github.com/bglmmz/grpc/credentials"
	"io/ioutil"
	"log"
)

func NewServerTLSOneWay(serverCertFile, serverKeyFile string) (credentials.TransportCredentials, error) {
	log.Printf("NewServerTLSOneWay")
	serverCert, err := tls.LoadX509KeyPair(serverCertFile, serverKeyFile)
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.NoClientCert,
	}), nil
}

func NewClientTLSOneWay(serverCaCertFile string) (credentials.TransportCredentials, error) {
	log.Printf("NewClientTLSOneWay")
	return credentials.NewTLS(&tls.Config{
		RootCAs:            loadCaPool(serverCaCertFile),
		InsecureSkipVerify: true,
	}), nil
}

func NewServerTLSTwoWay(clientCaCertFile, serverCertFile, serverKeyFile string) (credentials.TransportCredentials, error) {
	serverCert, err := tls.LoadX509KeyPair(serverCertFile, serverKeyFile)
	if err != nil {
		return nil, err
	}

	log.Printf("NewServerTLSTwoWay")
	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    loadCaPool(clientCaCertFile),
	}), nil
}

func NewClientTLSTwoWay(serverCaCertFile, clientCertFile, clientKeyFile string) (credentials.TransportCredentials, error) {
	log.Printf("NewClientTLSTwoWay")

	clientCert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(&tls.Config{
		Certificates:       []tls.Certificate{clientCert},
		RootCAs:            loadCaPool(serverCaCertFile),
		InsecureSkipVerify: true,
	}), nil
}

func NewServerGMTLSOneWay(serverSignCertFile, serverSignKeyFile, serverCipherCertFile, serverCipherKeyFile string) (credentials.TransportCredentials, error) {
	log.Printf("NewServerGMTLSOneWay")

	serverSignCert, err := tls.LoadX509KeyPair(serverSignCertFile, serverSignKeyFile)
	if err != nil {
		return nil, err
	}
	serverCipherCert, err := tls.LoadX509KeyPair(serverCipherCertFile, serverCipherKeyFile)
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{serverSignCert, serverCipherCert},
		GMSupport:    &tls.GMSupport{},
		ClientAuth:   tls.NoClientCert,
	}), nil
}

func NewClientGMTLSOneWay(serverCaCertFile string) (credentials.TransportCredentials, error) {
	log.Printf("NewClientGMTLSOneWay")
	return credentials.NewTLS(&tls.Config{
		GMSupport:          &tls.GMSupport{},
		RootCAs:            loadCaPool(serverCaCertFile),
		ClientAuth:         tls.NoClientCert,
		InsecureSkipVerify: true, //controls whether a client verifies the server's certificate chain and host name.
	}), nil
}

func NewServerGMTLSTwoWay(clientCaCertFile, serverSignCertFile, serverSignKeyFile, serverCipherCertFile, serverCipherKeyFile string) (credentials.TransportCredentials, error) {
	log.Printf("NewServerGMTLSTwoWay")

	serverSignCert, err := tls.LoadX509KeyPair(serverSignCertFile, serverSignKeyFile)
	if err != nil {
		return nil, err
	}

	serverCipherCert, err := tls.LoadX509KeyPair(serverCipherCertFile, serverCipherKeyFile)
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{serverSignCert, serverCipherCert},
		GMSupport:    &tls.GMSupport{},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    loadCaPool(clientCaCertFile),
	}), nil
}

func NewClientGMTLSTwoWay(serverCaCertFile, clientSignCertFile, clientSignKeyFile, clientCipherCertFile, clientCipherKeyFile string) (credentials.TransportCredentials, error) {
	log.Printf("NewClientTLSTwoWay")

	clientSignCert, err := tls.LoadX509KeyPair(clientSignCertFile, clientSignKeyFile)
	if err != nil {
		return nil, err
	}

	clientCipherCert, err := tls.LoadX509KeyPair(clientCipherCertFile, clientCipherKeyFile)
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(&tls.Config{
		Certificates:       []tls.Certificate{clientSignCert, clientCipherCert},
		GMSupport:          &tls.GMSupport{},
		RootCAs:            loadCaPool(serverCaCertFile),
		InsecureSkipVerify: true, //controls whether a client verifies the server's certificate chain and host name.
	}), nil
}

func loadCaPool(caCertFile string) *x509.CertPool {
	pemServerCA, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		log.Fatalf("failed to read CA's certificate %v", err)
	}

	cp := x509.NewCertPool()
	if !cp.AppendCertsFromPEM(pemServerCA) {
		log.Fatalf("failed to add CA's certificate.")
	}
	return cp
}
