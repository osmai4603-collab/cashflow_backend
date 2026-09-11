package signing

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"cashflow_backend/internal/domain/accounting"
)

const (
	dsNamespace    = "http://www.w3.org/2000/09/xmldsig#"
	xadesNamespace = "http://uri.etsi.org/01903/v1.3.2#"
)

type XAdESSigner struct{}

func (s *XAdESSigner) Sign(xmlContent []byte, cert *accounting.EDICertificate) ([]byte, error) {
	if cert == nil {
		return nil, fmt.Errorf("certificate is required")
	}
	if err := validateXML(xmlContent); err != nil {
		return nil, err
	}
	privateKey, err := parsePrivateKey(cert.PrivateKey)
	if err != nil {
		return nil, err
	}
	certificate, err := parseCertificate(cert.CertContent)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(bytes.TrimSpace(xmlContent))
	signedInfo := fmt.Sprintf(`<ds:SignedInfo xmlns:ds="%s"><ds:CanonicalizationMethod Algorithm="http://www.w3.org/2006/12/xml-c14n11"/><ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/><ds:Reference URI=""><ds:Transforms><ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/></ds:Transforms><ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/><ds:DigestValue>%s</ds:DigestValue></ds:Reference></ds:SignedInfo>`, dsNamespace, base64.StdEncoding.EncodeToString(digest[:]))
	signedInfoDigest := sha256.Sum256([]byte(signedInfo))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, signedInfoDigest[:])
	if err != nil {
		return nil, err
	}
	signatureID := "sig-" + base64.RawURLEncoding.EncodeToString(signature[:8])
	properties := time.Now().UTC().Format(time.RFC3339)
	block := fmt.Sprintf(`<ds:Signature xmlns:ds="%s" Id="%s">%s<ds:SignatureValue>%s</ds:SignatureValue><ds:KeyInfo><ds:X509Data><ds:X509Certificate>%s</ds:X509Certificate></ds:X509Data></ds:KeyInfo><xades:QualifyingProperties xmlns:xades="%s" Target="#%s"><xades:SignedProperties><xades:SignedSignatureProperties><xades:SigningTime>%s</xades:SigningTime><xades:SigningCertificate><xades:Cert><xades:CertDigest><ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/><ds:DigestValue>%s</ds:DigestValue></xades:CertDigest></xades:Cert></xades:SigningCertificate></xades:SignedSignatureProperties></xades:SignedProperties></xades:QualifyingProperties></ds:Signature>`, dsNamespace, signatureID, signedInfo, base64.StdEncoding.EncodeToString(signature), base64.StdEncoding.EncodeToString(certificate.Raw), xadesNamespace, signatureID, properties, certificateDigest(certificate))
	closing := bytes.LastIndex(xmlContent, []byte("</"))
	if closing < 0 {
		return nil, fmt.Errorf("XML root closing tag not found")
	}
	result := make([]byte, 0, len(xmlContent)+len(block))
	result = append(result, xmlContent[:closing]...)
	result = append(result, block...)
	result = append(result, xmlContent[closing:]...)
	return result, nil
}

func Verify(xmlContent []byte, certContent []byte) error {
	certificate, err := parseCertificate(certContent)
	if err != nil {
		return err
	}
	start := bytes.Index(xmlContent, []byte("<ds:SignedInfo"))
	end := bytes.Index(xmlContent, []byte("</ds:SignedInfo>"))
	if start < 0 || end < 0 {
		return fmt.Errorf("signature SignedInfo not found")
	}
	end += len("</ds:SignedInfo>")
	var signedInfo struct {
		DigestValue string `xml:"Reference>DigestValue"`
	}
	fragment := xmlContent[start:end]
	if err := xml.Unmarshal(fragment, &signedInfo); err != nil {
		return err
	}
	original := removeSignature(xmlContent)
	digest := sha256.Sum256(bytes.TrimSpace(original))
	if signedInfo.DigestValue != base64.StdEncoding.EncodeToString(digest[:]) {
		return fmt.Errorf("document digest mismatch")
	}
	signatureValueStart := bytes.Index(xmlContent, []byte("<ds:SignatureValue>"))
	signatureValueEnd := bytes.Index(xmlContent[signatureValueStart:], []byte("</ds:SignatureValue>"))
	if signatureValueStart < 0 || signatureValueEnd < 0 {
		return fmt.Errorf("signature value not found")
	}
	signatureValueStart += len("<ds:SignatureValue>")
	signatureValueEnd += signatureValueStart - len("<ds:SignatureValue>")
	signedInfoDigest := sha256.Sum256(fragment)
	signature, err := base64.StdEncoding.DecodeString(string(xmlContent[signatureValueStart:signatureValueEnd]))
	if err != nil {
		return err
	}
	return rsa.VerifyPKCS1v15(certificate.PublicKey.(*rsa.PublicKey), crypto.SHA256, signedInfoDigest[:], signature)
}

func removeSignature(content []byte) []byte {
	start := bytes.Index(content, []byte("<ds:Signature"))
	if start < 0 {
		return content
	}
	end := bytes.Index(content[start:], []byte("</ds:Signature>"))
	if end < 0 {
		return content
	}
	end += start + len("</ds:Signature>")
	return append(append([]byte{}, content[:start]...), content[end:]...)
}
func validateXML(content []byte) error {
	var value any
	if err := xml.Unmarshal(content, &value); err != nil {
		return fmt.Errorf("invalid XML: %w", err)
	}
	return nil
}
func parsePrivateKey(content []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(content)
	if block == nil {
		return nil, fmt.Errorf("private key PEM is required")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key must be RSA")
		}
		return rsaKey, nil
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
func parseCertificate(content []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(content)
	if block == nil {
		return nil, fmt.Errorf("certificate PEM is required")
	}
	return x509.ParseCertificate(block.Bytes)
}
func certificateDigest(cert *x509.Certificate) string {
	digest := sha256.Sum256(cert.Raw)
	return base64.StdEncoding.EncodeToString(digest[:])
}

var _ = strings.TrimSpace
