//
// Copyright (c) 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Initial implementation based on
//    golang/src/pkg/crypto/tls/generate_cert.go
//
// which is:
//
//    Copyright 2009 The Go Authors. All rights reserved.
//    Use of this source code is governed by a BSD-style
//    license that can be found in the golang LICENSE file.
//

package certs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"time"

	"github.com/01org/ciao/ssntp"
)

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) (*pem.Block, error) {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}, nil
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, fmt.Errorf("Unable to marshal ECDSA private key: %v", err)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}, nil
	default:
		return nil, fmt.Errorf("No private key found")
	}
}

func keyFromPemBlock(block *pem.Block) (interface{}, error) {
	if block.Type == "EC PRIVATE KEY" {
		return x509.ParseECPrivateKey(block.Bytes)
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func addOIDs(role ssntp.Role, oids []asn1.ObjectIdentifier) []asn1.ObjectIdentifier {
	if role.IsAgent() {
		oids = append(oids, ssntp.RoleAgentOID)
	}

	if role.IsScheduler() {
		oids = append(oids, ssntp.RoleSchedulerOID)
	}

	if role.IsController() {
		oids = append(oids, ssntp.RoleControllerOID)
	}

	if role.IsNetAgent() {
		oids = append(oids, ssntp.RoleNetAgentOID)
	}

	if role.IsServer() {
		oids = append(oids, ssntp.RoleServerOID)
	}

	if role.IsCNCIAgent() {
		oids = append(oids, ssntp.RoleCNCIAgentOID)
	}

	return oids
}

func generatePrivateKey(ell bool) (interface{}, error) {
	if ell == false {
		return rsa.GenerateKey(rand.Reader, 2048)
	}
	return ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
}

func addMgmtIPs(mgmtIPs []string, ips []net.IP) []net.IP {
	for _, i := range mgmtIPs {
		if ip := net.ParseIP(i); ip != nil {
			ips = append(ips, ip)
		}
	}

	return ips
}

func addDNSNames(hosts []string, names []string) []string {
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			continue
		} else {
			names = append(names, h)
		}
	}

	return names
}

// CreateCertTemplate provides the certificate template from which trust anchor or derivative certificates can be derived.
func CreateCertTemplate(role ssntp.Role, organization string, email string, hosts []string, mgmtIPs []string) (*x509.Certificate, error) {
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("Gailed to generate certificate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{organization},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		EmailAddresses:        []string{email},
		BasicConstraintsValid: true,
	}

	template.DNSNames = addDNSNames(hosts, template.DNSNames)
	template.IPAddresses = addMgmtIPs(mgmtIPs, template.IPAddresses)
	template.UnknownExtKeyUsage = addOIDs(role, template.UnknownExtKeyUsage)
	return &template, nil
}

// CreateAnchorCert creates the trust anchor certificate and the CA certificate. Both are written out PEM encoded.
func CreateAnchorCert(template *x509.Certificate, useElliptic bool, certOutput io.Writer, caCertOutput io.Writer) error {
	priv, err := generatePrivateKey(useElliptic)
	if err != nil {
		return fmt.Errorf("Unable to create private key: %v", err)
	}

	template.IsCA = true

	// Create self-signed certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, publicKey(priv), priv)
	if err != nil {
		return fmt.Errorf("Unable to create server certificate: %v", err)
	}

	// Write out CA cert
	err = pem.Encode(caCertOutput, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return fmt.Errorf("Unable to encode PEM block: %v", err)
	}

	// Write out certificate (including private key)
	err = pem.Encode(certOutput, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return fmt.Errorf("Unable to encode PEM block: %v", err)
	}
	block, err := pemBlockForKey(priv)
	if err != nil {
		return fmt.Errorf("Unable to get PEM block from key: %v", err)
	}
	err = pem.Encode(certOutput, block)
	if err != nil {
		return fmt.Errorf("Unable to encode PEM block: %v", err)
	}

	return nil
}

// CreateCert creates the certificate signed by the giver trust anchor certificate. It is written PEM encoded.
func CreateCert(template *x509.Certificate, useElliptic bool, anchorCert []byte, certOutput io.Writer) error {
	priv, err := generatePrivateKey(useElliptic)
	if err != nil {
		return fmt.Errorf("Unable to create private key: %v", err)
	}

	template.IsCA = false

	// Parent public key first
	certBlock, rest := pem.Decode(anchorCert)
	parentCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("Unable to parse anchor cert: %v", err)
	}

	// Parent private key
	privKeyBlock, _ := pem.Decode(rest)
	if privKeyBlock == nil {
		return fmt.Errorf("Unable to extract private key from anchor cert: %v", err)
	}

	anchorPrivKey, err := keyFromPemBlock(privKeyBlock)
	if err != nil {
		return fmt.Errorf("Unable to parse private key from anchor cert: %v", err)
	}

	// Create certificate signed by private key from anchorCert
	derBytes, err := x509.CreateCertificate(rand.Reader, template, parentCert, publicKey(priv), anchorPrivKey)
	if err != nil {
		return fmt.Errorf("Unable to create certificate: %v", err)
	}

	// Write out certificate (including private key)
	err = pem.Encode(certOutput, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return fmt.Errorf("Unable to encode PEM block: %v", err)
	}

	block, err := pemBlockForKey(priv)
	if err != nil {
		return fmt.Errorf("Unable to get PEM block from key: %v", err)
	}
	err = pem.Encode(certOutput, block)
	if err != nil {
		return fmt.Errorf("Unable to encode PEM block: %v", err)
	}

	return nil
}
