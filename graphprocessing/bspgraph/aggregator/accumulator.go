package aggregator

import (
	"math"
	"sync/atomic"
	"unsafe"
)

// Float64Accumulator implements a concurrent-safe accumulator for float64
type Float64Accumulator struct {
	prevSum float64
	curSum  float64
}

// Type implements bspgrah.Aggregator
func (a *Float64Accumulator) Type() string {
	return "Float64Accumulator"
}

// Get returns the current value of the accumulator
func (a *Float64Accumulator) Get() interface{} {
	return loadFloat64(&a.curSum)
}

// Set the current value of the accumulator
func (a *Float64Accumulator) Set(v interface{}) {
	for v64 := v.(float64); ; {
		oldCur := loadFloat64(&a.curSum)
		oldPrev := loadFloat64(&a.prevSum)
		swappedCur := atomic.CompareAndSwapUint64(
			(*uint64)(unsafe.Pointer(&a.curSum)),
			math.Float64bits(oldCur),
			math.Float64bits(v64),
		)
		swappedPrev := atomic.CompareAndSwapUint64(
			(*uint64)(unsafe.Pointer(&a.curSum)),
			math.Float64bits(oldPrev),
			math.Float64bits(v64),
		)
		if swappedCur && swappedPrev {
			return
		}
	}
}

// Aggregate adds a float64 value to the accumulator.
func (a *Float64Accumulator) Aggregate(v interface{}) {
	for v64 := v.(float64); ; {
		oldV := loadFloat64(&a.curSum)
		newV := oldV + v64
		if atomic.CompareAndSwapUint64(
			(*uint64)(unsafe.Pointer(&a.curSum)),
			math.Float64bits(oldV),
			math.Float64bits(newV),
		) {
			return
		}
	}
}

func loadFloat64(f *float64) float64 {
	return math.Float64frombits(atomic.LoadUint64((*uint64)(unsafe.Pointer(f))))
}
