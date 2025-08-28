package slices

import (
	"fmt"
	"slices"
	"testing"

	"github.com/deckhouse/sds-common-lib/utils/iter"
)

var benchSizes = []int{1_000, 10_000, 100_000, 1_000_000}

// global sinks to avoid compiler optimizations
var (
	benchResultIndexInt    int
	benchResultIndexDevice int
	benchResultIntPtr      *int
	benchResultDevicePtr   *Device
)

func makeInts(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = i
	}
	return s
}

func BenchmarkPtrs_Miss(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			data := makeInts(n)
			b.ResetTimer()
			for b.Loop() {
				benchResultIntPtr, _ = iter.Find(Ptrs(data), func(v *int) bool { return *v == -1 })
			}
		})
	}
}

func BenchmarkIndexFunc_Miss(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			data := makeInts(n)
			b.ResetTimer()
			for b.Loop() {
				benchResultIndexInt = slices.IndexFunc(data, func(v int) bool { return v == -1 })
			}
		})
	}
}

type Device struct {
	Volume       int
	Minor        int
	DiskState    string
	Client       bool
	Open         bool
	Quorum       bool
	Size         int
	Read         int
	Written      int
	ALWrites     int
	BMWrites     int
	UpperPending int
	LowerPending int
}

func makeDevices(n int) []Device {
	s := make([]Device, n)
	return s
}

// Miss-only benchmarks with zero-valued Device entries across sizes
func BenchmarkPtrs_DeviceMiss(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			data := makeDevices(n)
			b.ResetTimer()
			for b.Loop() {
				benchResultDevicePtr, _ = iter.Find(Ptrs(data), func(v *Device) bool { return v.Volume == -1 })
			}
		})
	}
}

func BenchmarkIndexFunc_DeviceMiss(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			data := makeDevices(n)
			b.ResetTimer()
			for b.Loop() {
				benchResultIndexDevice = slices.IndexFunc(data, func(v Device) bool { return v.Volume == -1 })
			}
		})
	}
}
