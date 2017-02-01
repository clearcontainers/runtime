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

package certs

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"net"
	"reflect"
	"testing"

	"github.com/01org/ciao/ssntp"
)

func TestGenerateKey(t *testing.T) {
	key, err := generatePrivateKey(false)

	_, ok := key.(*rsa.PrivateKey)
	if !ok || err != nil {
		t.Errorf("RSA key expected from generatePrivateKey: %v", err)
	}

	pKey := publicKey(key)
	_, ok = pKey.(*rsa.PublicKey)
	if !ok {
		t.Error("RSA public key expected")
	}

	key, err = generatePrivateKey(true)
	_, ok = key.(*ecdsa.PrivateKey)

	if !ok || err != nil {
		t.Errorf("ECDSA key expected from generatePrivateKey: %v", err)
	}

	pKey = publicKey(key)
	_, ok = pKey.(*ecdsa.PublicKey)
	if !ok {
		t.Error("ECDSA public key expected")
	}
}

func TestPemBlockParsing(t *testing.T) {
	origKey, _ := generatePrivateKey(false)

	block, err := pemBlockForKey(origKey)

	if block.Type != "RSA PRIVATE KEY" || err != nil {
		t.Errorf("Unexpected PEM block type: %v: err: %v", block.Type, err)
	}

	parsedKey, err := keyFromPemBlock(block)
	_, ok := parsedKey.(*rsa.PrivateKey)

	if !ok || err != nil {
		t.Errorf("Expected RSA private key: %v", err)
	}

	origKey, _ = generatePrivateKey(true)

	block, err = pemBlockForKey(origKey)

	if block.Type != "EC PRIVATE KEY" || err != nil {
		t.Errorf("Unexpected PEM block type: %v: err: %v", block.Type, err)
	}

	parsedKey, err = keyFromPemBlock(block)
	_, ok = parsedKey.(*ecdsa.PrivateKey)

	if !ok || err != nil {
		t.Errorf("Expected ECDSA private key: %v", err)
	}
}

func TestAddOIDs(t *testing.T) {
	mapping := map[ssntp.Role]asn1.ObjectIdentifier{
		ssntp.AGENT:      ssntp.RoleAgentOID,
		ssntp.SCHEDULER:  ssntp.RoleSchedulerOID,
		ssntp.Controller: ssntp.RoleControllerOID,
		ssntp.NETAGENT:   ssntp.RoleNetAgentOID,
		ssntp.SERVER:     ssntp.RoleServerOID,
		ssntp.CNCIAGENT:  ssntp.RoleCNCIAgentOID,
	}

	// Check all
	for role, oid := range mapping {
		oids := addOIDs(role, []asn1.ObjectIdentifier{})
		if len(oids) != 1 {
			t.Errorf("Expected only one OID found %d", len(oids))
		}

		if !reflect.DeepEqual(oids[0], oid) {
			t.Errorf("Unexpected role to OID mapping: %v -> %v", role, oid)
		}
	}

	// Check common pairing
	oids := addOIDs(ssntp.AGENT|ssntp.NETAGENT, []asn1.ObjectIdentifier{})
	if len(oids) != 2 {
		t.Errorf("Expected two OIDS found %d", len(oids))
	}

	if !reflect.DeepEqual(oids[0], ssntp.RoleAgentOID) || !reflect.DeepEqual(oids[1], ssntp.RoleNetAgentOID) {
		t.Errorf("Unexpected OIDS in list: %v and %v", oids[0], oids[1])
	}
}

func TestCreateCertTemplateHosts(t *testing.T) {
	hosts := []string{"test.example.com", "test2.example.com"}
	mgmtIPs := []string{}
	cert, err := CreateCertTemplate(ssntp.AGENT, "ACME Corp", "test@example.com", hosts, mgmtIPs)

	if err != nil {
		t.Errorf("Unexpected error when creating cert template: %v", err)
	}

	if !reflect.DeepEqual(cert.DNSNames, hosts) {
		t.Errorf("Hosts in certificate don't match: %v %v", hosts, cert.DNSNames)
	}
}

func TestCreateCertTemplateIPs(t *testing.T) {
	hosts := []string{}
	mgmtIPs := []string{"127.0.0.1", "127.0.0.2"}
	cert, err := CreateCertTemplate(ssntp.AGENT, "ACME Corp", "test@example.com", hosts, mgmtIPs)

	if err != nil {
		t.Errorf("Unexpected error when creating cert template: %v", err)
	}

	IPs := []net.IP{net.IPv4(127, 0, 0, 1), net.IPv4(127, 0, 0, 2)}
	if !reflect.DeepEqual(cert.IPAddresses, IPs) {
		t.Errorf("IPs in certificate don't match: %v %v", hosts, cert.DNSNames)
	}
}

func TestCreateCertTemplateRoles(t *testing.T) {
	hosts := []string{"test.example.com", "test2.example.com"}
	mgmtIPs := []string{}
	cert, err := CreateCertTemplate(ssntp.AGENT|ssntp.NETAGENT, "ACME Corp", "test@example.com", hosts, mgmtIPs)

	if err != nil {
		t.Errorf("Unexpected error when creating cert template: %v", err)
	}

	if ssntp.GetRoleFromOIDs(cert.UnknownExtKeyUsage) != ssntp.AGENT|ssntp.NETAGENT {
		t.Error("Roles on created cert do not match those requested")
	}
}

func TestCreateAnchorCert(t *testing.T) {
	var certOutput, caCertOutput bytes.Buffer

	hosts := []string{"test.example.com", "test2.example.com"}
	mgmtIPs := []string{}

	template, err := CreateCertTemplate(ssntp.AGENT, "ACME Corp", "test@example.com", hosts, mgmtIPs)
	if err != nil {
		t.Errorf("Unexpected error when creating cert template: %v", err)
	}

	err = CreateAnchorCert(template, false, &certOutput, &caCertOutput)
	if err != nil {
		t.Errorf("Unexpected error when creating anchor cert: %v", err)
	}

	// Decode server cert & private key
	certBlock, rest := pem.Decode(certOutput.Bytes())
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		t.Errorf("Failed to parse certificate: %v", err)
	}

	if cert.IsCA != true {
		t.Errorf("Expected certificate to be a CA")
	}

	privKeyBlock, _ := pem.Decode(rest)
	if privKeyBlock == nil {
		t.Errorf("Unable to extract private key from anchor cert")
	}

	anchorPrivKey, err := keyFromPemBlock(privKeyBlock)
	if err != nil {
		t.Errorf("Unable to parse private key from anchor cert: %v", err)
	}

	_, ok := anchorPrivKey.(*rsa.PrivateKey)
	if !ok || err != nil {
		t.Errorf("Expected RSA private key: %v", err)
	}

	// Decode CA
	certBlock, rest = pem.Decode(caCertOutput.Bytes())
	if len(rest) > 0 {
		t.Error("Unexpected data after certificate PEM block")
	}

	cert, err = x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		t.Errorf("Failed to parse certificate: %v", err)
	}

	if cert.IsCA != true {
		t.Errorf("Expected certificate to be a CA")
	}
}

func TestCreateCert(t *testing.T) {
	var anchorCertOutput, caCertOutput, certOutput bytes.Buffer

	hosts := []string{"test.example.com", "test2.example.com"}
	mgmtIPs := []string{}

	template, err := CreateCertTemplate(ssntp.AGENT, "ACME Corp", "test@example.com", hosts, mgmtIPs)
	if err != nil {
		t.Errorf("Unexpected error when creating cert template: %v", err)
	}

	err = CreateAnchorCert(template, false, &anchorCertOutput, &caCertOutput)
	if err != nil {
		t.Errorf("Unexpected error when creating anchor cert: %v", err)
	}

	err = CreateCert(template, false, anchorCertOutput.Bytes(), &certOutput)
	if err != nil {
		t.Errorf("Unexpected error when creating signed cert: %v", err)
	}

	// Decode signed cert & private key
	certBlock, rest := pem.Decode(certOutput.Bytes())
	privKeyBlock, _ := pem.Decode(rest)
	if privKeyBlock == nil {
		t.Errorf("Unable to extract private key from anchor cert")
	}

	anchorPrivKey, err := keyFromPemBlock(privKeyBlock)
	if err != nil {
		t.Errorf("Unable to parse private key from anchor cert: %v", err)
	}

	_, ok := anchorPrivKey.(*rsa.PrivateKey)
	if !ok || err != nil {
		t.Errorf("Expected RSA private key: %v", err)
	}

	// Verify cert is signed with CA cert
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		t.Errorf("Failed to parse certificate: %v", err)
	}

	if cert.IsCA != false {
		t.Errorf("Expected certificate not to be a CA")
	}

	roots := x509.NewCertPool()
	ok = roots.AppendCertsFromPEM(caCertOutput.Bytes())
	if !ok {
		t.Errorf("Could not add CA cert to pool")
	}

	opts := x509.VerifyOptions{
		Roots: roots,
	}

	if _, err = cert.Verify(opts); err != nil {
		t.Errorf("Failed to verify certificate: %v", err)
	}
}
