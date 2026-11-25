// Package capacity holds the type a storage class of a Deckhouse storage
// module publishes in status.capacity, and the rules for writing it.
//
// A module embeds [Capacity] inline into its own capacity type and adds the
// fields only it can report, such as the pool name or per-pool figures:
//
//	type CephStorageClassCapacity struct {
//		capacity.Capacity `json:",inline"`
//		Pool string `json:"pool"`
//	}
//
// The console reads the fields of [Capacity] by name from any class, so a
// module keeps their meaning and leaves out a figure it cannot compute rather
// than publishing zero.
package capacity

import (
	"math"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Granularity is the step [RoundDown] and [Quantity] round to: 1 MiB.
	Granularity int64 = 1 << 20
	// RefreshPeriod is the longest time [NeedsWrite] keeps unchanged figures
	// before it asks for a write that renews UpdatedAt.
	RefreshPeriod = 5 * time.Minute
	// ChangeShare is the share of the size a figure has to move by before
	// [NeedsWrite] asks for a write: 1%.
	ChangeShare = 0.01
)

// Capacity is the space of the storage a class provisions in.
// +k8s:deepcopy-gen=true
type Capacity struct {
	// Total size of the storage the class provisions in.
	Size resource.Quantity `json:"size"`
	// Data written into the storage, including the data of snapshots and of deleted volumes the storage still keeps.
	Used resource.Quantity `json:"used"`
	// Space the storage can still take before it is full.
	FreeOnDisk resource.Quantity `json:"freeOnDisk"`
	// Sum of the provisioned sizes of the volumes and snapshots of the class. Absent when the module does not count it.
	// +optional
	Allocated *resource.Quantity `json:"allocated,omitempty"`
	// Quota set on the storage. Absent when the storage has none.
	// +optional
	Quota *resource.Quantity `json:"quota,omitempty"`
	// Space the storage can still take within its quota. Absent when the storage has no quota.
	// +optional
	FreeByQuota *resource.Quantity `json:"freeByQuota,omitempty"`
	// Space new volumes can still be given within the overprovisioning limit of the storage. Absent when the storage has no such limit.
	// +optional
	FreeByAllocationLimit *resource.Quantity `json:"freeByAllocationLimit,omitempty"`
	// Data only snapshots hold, freed when they are deleted. Part of `used`. Absent when the module cannot measure it.
	// +optional
	HeldBySnapshots *resource.Quantity `json:"heldBySnapshots,omitempty"`
	// Data of volumes deleted in Kubernetes that the storage still keeps, such as images in a trash or volumes kept for their snapshots. Part of `used`. Absent when the module cannot measure it.
	// +optional
	HeldByDeleted *resource.Quantity `json:"heldByDeleted,omitempty"`
	// When the figures were read from the storage.
	UpdatedAt metav1.Time `json:"updatedAt"`
}

// RoundDown rounds a number of bytes down to [Granularity]. A negative number
// gives zero.
func RoundDown(bytes int64) int64 {
	bytes = max(bytes, 0)
	return bytes - bytes%Granularity
}

// Quantity returns bytes rounded down by [RoundDown] as a binary quantity.
func Quantity(bytes int64) resource.Quantity {
	return *resource.NewQuantity(RoundDown(bytes), resource.BinarySI)
}

// OptionalQuantity is [Quantity] for an optional field.
func OptionalQuantity(bytes int64) *resource.Quantity {
	q := Quantity(bytes)
	return &q
}

// NeedsWrite reports whether next has to replace prev in the status. It does
// when one of them is absent and the other is not, when prev was written
// [RefreshPeriod] or longer before now or after now, when an optional figure
// appears or disappears, when the quota changes, and when a figure moves by
// more than [ChangeShare] of the size in next.
//
// A module with fields of its own checks them after NeedsWrite, comparing
// figures with [Moved].
func NeedsWrite(prev, next *Capacity, now time.Time) bool {
	if prev == nil || next == nil {
		return prev != next
	}
	if elapsed := now.Sub(prev.UpdatedAt.Time); elapsed < 0 || elapsed >= RefreshPeriod {
		return true
	}
	if !sameQuantity(prev.Quota, next.Quota) {
		return true
	}
	if Moved(prev.Size, next.Size, next.Size) ||
		Moved(prev.Used, next.Used, next.Size) ||
		Moved(prev.FreeOnDisk, next.FreeOnDisk, next.Size) {
		return true
	}
	for _, pair := range [][2]*resource.Quantity{
		{prev.Allocated, next.Allocated},
		{prev.FreeByQuota, next.FreeByQuota},
		{prev.FreeByAllocationLimit, next.FreeByAllocationLimit},
		{prev.HeldBySnapshots, next.HeldBySnapshots},
		{prev.HeldByDeleted, next.HeldByDeleted},
	} {
		if MovedOptional(pair[0], pair[1], next.Size) {
			return true
		}
	}
	return false
}

// Moved reports whether next differs from prev by more than [ChangeShare] of
// of. Any change counts when of is zero.
func Moved(prev, next, of resource.Quantity) bool {
	delta := math.Abs(float64(next.Value() - prev.Value()))
	if delta == 0 {
		return false
	}
	return delta > ChangeShare*float64(of.Value())
}

// MovedOptional is [Moved] for an optional figure: it also reports true when
// the figure appears or disappears.
func MovedOptional(prev, next *resource.Quantity, of resource.Quantity) bool {
	if prev == nil || next == nil {
		return (prev == nil) != (next == nil)
	}
	return Moved(*prev, *next, of)
}

func sameQuantity(a, b *resource.Quantity) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Cmp(*b) == 0
}
