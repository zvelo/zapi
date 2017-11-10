package httpsig

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/asn1"
	"hash"
	"math/big"
	"strconv"

	"github.com/pkg/errors"
)

// Algorithm represents the type of HTTP signature to use
type Algorithm int

// These are the available Algorithms to use
const (
	Unknown Algorithm = iota
	RSASHA1
	RSASHA256
	HMACSHA256
	ECDSASHA256
)

func (a Algorithm) String() string {
	switch a {
	case RSASHA1:
		return "rsa-sha1"
	case RSASHA256:
		return "rsa-sha256"
	case HMACSHA256:
		return "hmac-sha256"
	case ECDSASHA256:
		return "ecdsa-sha256"
	}
	return "Algorithm(" + strconv.Itoa(int(a)) + ")"
}

// ParseAlgorithm parses a string into an Algorithm
func ParseAlgorithm(val string) Algorithm {
	switch val {
	case RSASHA1.String():
		return RSASHA1
	case RSASHA256.String():
		return RSASHA256
	case HMACSHA256.String():
		return HMACSHA256
	case ECDSASHA256.String():
		return ECDSASHA256
	}
	return Unknown
}

func rsaSign(key interface{}, hash crypto.Hash, hashed []byte) ([]byte, error) {
	var priv *rsa.PrivateKey

	switch k := key.(type) {
	case rsa.PrivateKey:
		priv = &k
	case *rsa.PrivateKey:
		priv = k
	default:
		return nil, errors.Errorf("invalid key type %T for RSA", key)
	}

	return rsa.SignPKCS1v15(rand.Reader, priv, hash, hashed)
}

func rsaVerify(key interface{}, hash crypto.Hash, hashed, sig []byte) error {
	var pub *rsa.PublicKey

	switch k := key.(type) {
	case rsa.PrivateKey:
		pub = &k.PublicKey
	case *rsa.PrivateKey:
		pub = &k.PublicKey
	case rsa.PublicKey:
		pub = &k
	case *rsa.PublicKey:
		pub = k
	default:
		return errors.Errorf("invalid key type %T for RSA", key)
	}

	return rsa.VerifyPKCS1v15(pub, hash, hashed, sig)
}

func hmacSign(h func() hash.Hash, key interface{}, data []byte) ([]byte, error) {
	k, ok := key.([]byte)
	if !ok {
		return nil, errors.Errorf("invalid key type %T for HMAC", key)
	}

	hash := hmac.New(h, k)
	if _, err := hash.Write(data); err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

func hmacVerify(h func() hash.Hash, key interface{}, data, sig []byte) error {
	k, ok := key.([]byte)
	if !ok {
		return errors.Errorf("invalid key type %T for HMAC", key)
	}

	hash := hmac.New(h, k)
	if _, err := hash.Write(data); err != nil {
		return err
	}

	if !hmac.Equal(hash.Sum(nil), sig) {
		return errors.New("invalid hmac")
	}

	return nil
}

type ecdsaSignature struct {
	R, S *big.Int
}

func ecdsaSign(key interface{}, hashed []byte) ([]byte, error) {
	var priv *ecdsa.PrivateKey

	switch k := key.(type) {
	case ecdsa.PrivateKey:
		priv = &k
	case *ecdsa.PrivateKey:
		priv = k
	default:
		return nil, errors.Errorf("invalid key type %T for ECDSA", key)
	}

	r, s, err := ecdsa.Sign(rand.Reader, priv, hashed[:])
	if err != nil {
		return nil, err
	}

	return asn1.Marshal(ecdsaSignature{r, s})
}

func ecdsaVerify(key interface{}, hashed, sig []byte) error {
	var esig ecdsaSignature
	if _, err := asn1.Unmarshal(sig, &esig); err != nil {
		return err
	}

	var pub *ecdsa.PublicKey

	switch k := key.(type) {
	case ecdsa.PrivateKey:
		pub = &k.PublicKey
	case *ecdsa.PrivateKey:
		pub = &k.PublicKey
	case ecdsa.PublicKey:
		pub = &k
	case *ecdsa.PublicKey:
		pub = k
	default:
		return errors.Errorf("invalid key type %T for ECDSA", key)
	}

	if !ecdsa.Verify(pub, hashed, esig.R, esig.S) {
		return errors.New("invalid ecdsa signature")
	}

	return nil
}

// Sign signs the data with the provided key. key is expected to be an
// rsa.PrivateKey, []byte for HMAC or ecdsa.PrivateKey
func (a Algorithm) Sign(key interface{}, data []byte) ([]byte, error) {
	switch a {
	case RSASHA1:
		hashed := sha1.Sum(data)
		return rsaSign(key, crypto.SHA1, hashed[:])
	case RSASHA256:
		hashed := sha256.Sum256(data)
		return rsaSign(key, crypto.SHA256, hashed[:])
	case HMACSHA256:
		return hmacSign(sha256.New, key, data)
	case ECDSASHA256:
		hashed := sha256.Sum256(data)
		return ecdsaSign(key, hashed[:])
	default:
		return nil, errors.Errorf("unsupported algorithm %s", a)
	}
}

// Verify verifies that data was properly signed by key. sig is the already
// signed data. key is expected to be an rsa.PublicKey, []byte for HMAC or
// ecdsa.PublicKey. PrivateKeys may also be used.
func (a Algorithm) Verify(key interface{}, data, sig []byte) error {
	switch a {
	case RSASHA1:
		hashed := sha1.Sum(data)
		return rsaVerify(key, crypto.SHA1, hashed[:], sig)
	case RSASHA256:
		hashed := sha256.Sum256(data)
		return rsaVerify(key, crypto.SHA256, hashed[:], sig)
	case HMACSHA256:
		return hmacVerify(sha256.New, key, data, sig)
	case ECDSASHA256:
		hashed := sha256.Sum256(data)
		return ecdsaVerify(key, hashed[:], sig)
	default:
		return errors.Errorf("unsupported algorithm %s", a)
	}
}
