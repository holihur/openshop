package signing

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// The Chinese providers sign requests with RSA rather than a shared secret:
// Alipay uses RSA2 (SHA-256 with RSA) over a sorted parameter string, and WeChat
// uses SHA-256 with RSA over the request line. These helpers are shared by the
// payment and identity adapters, so signing and verification cannot drift apart
// between the two.

// parseKeyMaterial accepts a PEM block or a bare base64 body, because
// environment variables are often populated with a single line.
func parseKeyMaterial(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("empty key")
	}
	if strings.Contains(raw, "-----BEGIN") {
		return []byte(raw), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(raw, " ", ""))
	if err != nil {
		return nil, fmt.Errorf("key is neither PEM nor base64: %w", err)
	}
	return decoded, nil
}

// ParseRSAPrivateKey reads a PKCS#1 or PKCS#8 RSA private key.
func ParseRSAPrivateKey(raw string) (*rsa.PrivateKey, error) {
	body, err := parseKeyMaterial(raw)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(body)
	if block == nil {
		return nil, errors.New("no PEM block found in private key")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return key, nil
}

// ParseRSAPublicKey reads a PKIX (SubjectPublicKeyInfo) or PKCS#1 RSA public key.
func ParseRSAPublicKey(raw string) (*rsa.PublicKey, error) {
	body, err := parseKeyMaterial(raw)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(body)
	if block == nil {
		return nil, errors.New("no PEM block found in public key")
	}
	if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
		key, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("certificate does not carry an RSA key")
		}
		return key, nil
	}
	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key is not RSA")
		}
		return rsaKey, nil
	}
	key, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	return key, nil
}

// signSHA256RSA returns the base64 signature of message.
func SignSHA256RSA(key *rsa.PrivateKey, message string) (string, error) {
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

// verifySHA256RSA checks a base64 signature over message.
func VerifySHA256RSA(key *rsa.PublicKey, message, signature string) error {
	decoded, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("signature is not base64: %w", err)
	}
	digest := sha256.Sum256([]byte(message))
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], decoded)
}

// alipaySignatureBase builds the string Alipay signs and verifies: every
// parameter except sign and sign_type, with empty values dropped, sorted by name
// and joined as k=v&k=v. Both directions use this one function, so signing and
// verification cannot drift apart.
func AlipaySignatureBase(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if key == "sign" || key == "sign_type" || strings.TrimSpace(value) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, key := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(params[key])
	}
	return b.String()
}
