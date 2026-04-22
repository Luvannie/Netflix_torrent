//go:build windows

package config

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	crypt32                = syscall.NewLazyDLL("Crypt32.dll")
	procCryptProtectData   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
	kernel32               = syscall.NewLazyDLL("Kernel32.dll")
	procLocalFree          = kernel32.NewProc("LocalFree")
)

type dpapiProtector struct{}

func newDefaultSecretProtector() SecretProtector {
	return dpapiProtector{}
}

type dataBlob struct {
	cbData uint32
	pbData *byte
}

func (dpapiProtector) Protect(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}

	in := dataBlob{
		cbData: uint32(len(data)),
		pbData: &data[0],
	}
	var out dataBlob

	r1, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&out)),
	)
	if r1 == 0 {
		return nil, fmt.Errorf("CryptProtectData failed: %w", err)
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))

	return append([]byte(nil), unsafe.Slice(out.pbData, int(out.cbData))...), nil
}

func (dpapiProtector) Unprotect(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}

	in := dataBlob{
		cbData: uint32(len(data)),
		pbData: &data[0],
	}
	var out dataBlob

	r1, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&out)),
	)
	if r1 == 0 {
		return nil, fmt.Errorf("CryptUnprotectData failed: %w", err)
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))

	return append([]byte(nil), unsafe.Slice(out.pbData, int(out.cbData))...), nil
}
