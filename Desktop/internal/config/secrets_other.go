//go:build !windows

package config

import "encoding/base64"

type base64Protector struct{}

func newDefaultSecretProtector() SecretProtector {
	return base64Protector{}
}

func (base64Protector) Protect(data []byte) ([]byte, error) {
	encoded := base64.StdEncoding.EncodeToString(data)
	return []byte(encoded), nil
}

func (base64Protector) Unprotect(data []byte) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}
	return decoded, nil
}
