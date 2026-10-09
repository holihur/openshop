package signing

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"
	"testing"
)

func keyPair(t *testing.T) (string, string, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	priv := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	return string(priv), string(pub), key
}

func TestSignVerifyRoundTrip(t *testing.T) {
	privPEM, _, key := keyPair(t)
	priv, err := ParseRSAPrivateKey(privPEM)
	if err != nil {
		t.Fatalf("private key: %v", err)
	}
	pub, err := ParseRSAPublicKey(string(pem.EncodeToMemory(&pem.Block{
		Type: "PUBLIC KEY", Bytes: mustPKIX(t, &key.PublicKey),
	})))
	if err != nil {
		t.Fatalf("public key: %v", err)
	}
	signature, err := SignSHA256RSA(priv, "hello")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if err := VerifySHA256RSA(pub, "hello", signature); err != nil {
		t.Fatalf("verify: %v", err)
	}
	// A different message, a truncated signature and garbage must all fail.
	if err := VerifySHA256RSA(pub, "hello!", signature); err == nil {
		t.Error("a different message must not verify")
	}
	if err := VerifySHA256RSA(pub, "hello", signature[:20]); err == nil {
		t.Error("a truncated signature must not verify")
	}
	if err := VerifySHA256RSA(pub, "hello", "!!!"); err == nil {
		t.Error("a non-base64 signature must not verify")
	}
	_ = privPEM
}

func TestAlipaySignatureBase(t *testing.T) {
	base := AlipaySignatureBase(map[string]string{
		"app_id": "1", "sign": "ignored", "sign_type": "RSA2",
		"method": "alipay.trade.page.pay", "empty": "", "biz_content": "{}",
	})
	// Sorted by name, empties and sign fields removed.
	want := "app_id=1&biz_content={}&method=alipay.trade.page.pay"
	if base != want {
		t.Errorf("base = %q, want %q", base, want)
	}
	if strings.Contains(base, "sign") {
		t.Error("sign fields must not be part of the signature base")
	}
}

func TestParseKeyMaterialRejectsGarbage(t *testing.T) {
	if _, err := ParseRSAPrivateKey(""); err == nil {
		t.Error("empty input must fail")
	}
	if _, err := ParseRSAPublicKey("bm90IGEga2V5"); err == nil {
		t.Error("non-PEM base64 must fail")
	}
}

func TestParseAcceptsPKCS8AndBase64Public(t *testing.T) {
	_, pubPEM, key := keyPair(t)
	pkcs8, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal pkcs8: %v", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8})
	if _, err := ParseRSAPrivateKey(string(privPEM)); err != nil {
		t.Fatalf("PKCS#8 private key: %v", err)
	}
	// A base64 body without PEM headers is accepted, because environment
	// variables are often single-line.
	body, _ := pem.Decode([]byte(pubPEM))
	encoded := base64.StdEncoding.EncodeToString(pem.EncodeToMemory(body))
	if _, err := ParseRSAPublicKey(encoded); err != nil {
		t.Fatalf("base64 public key: %v", err)
	}
}

func mustPKIX(t *testing.T, key *rsa.PublicKey) []byte {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	return der
}
