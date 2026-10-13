//go:build windows

package mbvip

import (
	"math"
	"syscall"
)

func requireProc(proc *syscall.LazyProc, name string) error {
	if proc.Find() != nil {
		return ProcNotFoundError{Name: name}
	}
	return nil
}

func boolToUintptr(v bool) uintptr {
	if v {
		return 1
	}
	return 0
}

func toCString(text string) []byte {
	return append([]byte(text), 0)
}

func toUTF16(text string) []uint16 {
	b, err := syscall.UTF16FromString(text)
	if err != nil {
		return []uint16{0}
	}
	return b
}

func float32ToUintptr(v float32) uintptr {
	return uintptr(math.Float32bits(v))
}

func uintptrToFloat32(v uintptr) float32 {
	return math.Float32frombits(uint32(v))
}

func uintptrToFloat64(v uintptr) float64 {
	return math.Float64frombits(uint64(v))
}
