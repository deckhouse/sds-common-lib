package capacity_test

import (
	"encoding/json"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/deckhouse/sds-common-lib/capacity"
)

const (
	mib = int64(1) << 20
	gib = int64(1) << 30
)

func TestRoundDown(t *testing.T) {
	for _, tc := range []struct {
		in, want int64
	}{
		{0, 0},
		{-1, 0},
		{-5 * mib, 0},
		{1, 0},
		{mib - 1, 0},
		{mib, mib},
		{mib + 1, mib},
		{10*gib + 3*mib + 512, 10*gib + 3*mib},
	} {
		if got := capacity.RoundDown(tc.in); got != tc.want {
			t.Errorf("RoundDown(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestQuantityIsBinaryAndRounded(t *testing.T) {
	q := capacity.Quantity(10*gib + 123)
	if q.Value() != 10*gib {
		t.Fatalf("value = %d, want %d", q.Value(), 10*gib)
	}
	if q.Format != resource.BinarySI {
		t.Fatalf("format = %s, want %s", q.Format, resource.BinarySI)
	}
	if q.String() != "10Gi" {
		t.Fatalf("string = %s, want 10Gi", q.String())
	}

	p := capacity.OptionalQuantity(3*mib + 1)
	if p == nil || p.Value() != 3*mib || p.String() != "3Mi" {
		t.Fatalf("OptionalQuantity = %v, want 3Mi", p)
	}
}

func TestMoved(t *testing.T) {
	size := resource.MustParse("100Gi")
	for _, tc := range []struct {
		name       string
		prev, next string
		of         resource.Quantity
		want       bool
	}{
		{"same", "10Gi", "10Gi", size, false},
		{"less than one percent", "10Gi", "11Gi", size, false},
		{"more than one percent up", "10Gi", "12Gi", size, true},
		{"more than one percent down", "12Gi", "10Gi", size, true},
		{"any change of a zero size", "0", "1Mi", resource.Quantity{}, true},
		{"no change of a zero size", "0", "0", resource.Quantity{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := capacity.Moved(resource.MustParse(tc.prev), resource.MustParse(tc.next), tc.of); got != tc.want {
				t.Fatalf("Moved(%s, %s) = %v, want %v", tc.prev, tc.next, got, tc.want)
			}
		})
	}
}

func TestMovedOptional(t *testing.T) {
	size := resource.MustParse("100Gi")
	small := resource.MustParse("1Gi")
	big := resource.MustParse("5Gi")

	if capacity.MovedOptional(nil, nil, size) {
		t.Error("two absent figures moved")
	}
	if !capacity.MovedOptional(nil, &small, size) {
		t.Error("an appearing figure did not move")
	}
	if !capacity.MovedOptional(&small, nil, size) {
		t.Error("a disappearing figure did not move")
	}
	if capacity.MovedOptional(&small, &small, size) {
		t.Error("an unchanged figure moved")
	}
	if !capacity.MovedOptional(&small, &big, size) {
		t.Error("a figure that grew by 4% of the size did not move")
	}
}

func written(at time.Time) *capacity.Capacity {
	return &capacity.Capacity{
		Size:       capacity.Quantity(100 * gib),
		Used:       capacity.Quantity(40 * gib),
		FreeOnDisk: capacity.Quantity(60 * gib),
		UpdatedAt:  metav1.NewTime(at),
	}
}

func TestNeedsWrite(t *testing.T) {
	at := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	now := at.Add(time.Minute)

	for _, tc := range []struct {
		name   string
		prev   *capacity.Capacity
		change func(next *capacity.Capacity)
		now    time.Time
		want   bool
	}{
		{name: "nothing was written", prev: nil, want: true},
		{name: "nothing changed", prev: written(at), want: false},
		{name: "small change", prev: written(at), change: func(n *capacity.Capacity) {
			n.Used = capacity.Quantity(40*gib + 512*mib)
			n.FreeOnDisk = capacity.Quantity(60*gib - 512*mib)
		}, want: false},
		{name: "used moved by more than one percent", prev: written(at), change: func(n *capacity.Capacity) {
			n.Used = capacity.Quantity(42 * gib)
		}, want: true},
		{name: "free moved by more than one percent", prev: written(at), change: func(n *capacity.Capacity) {
			n.FreeOnDisk = capacity.Quantity(58 * gib)
		}, want: true},
		{name: "size moved by more than one percent", prev: written(at), change: func(n *capacity.Capacity) {
			n.Size = capacity.Quantity(110 * gib)
		}, want: true},
		{name: "refresh period passed", prev: written(at), now: at.Add(capacity.RefreshPeriod), want: true},
		{name: "refresh period not yet passed", prev: written(at), now: at.Add(capacity.RefreshPeriod - time.Second), want: false},
		{name: "written after now", prev: written(at), now: at.Add(-time.Second), want: true},
		{name: "quota set", prev: written(at), change: func(n *capacity.Capacity) {
			n.Quota = capacity.OptionalQuantity(200 * gib)
		}, want: true},
		{name: "quota changed by little", prev: func() *capacity.Capacity {
			p := written(at)
			p.Quota = capacity.OptionalQuantity(200 * gib)
			return p
		}(), change: func(n *capacity.Capacity) {
			n.Quota = capacity.OptionalQuantity(200*gib + mib)
		}, want: true},
		{name: "allocated appeared", prev: written(at), change: func(n *capacity.Capacity) {
			n.Allocated = capacity.OptionalQuantity(0)
		}, want: true},
		{name: "held by deleted disappeared", prev: func() *capacity.Capacity {
			p := written(at)
			p.HeldByDeleted = capacity.OptionalQuantity(gib)
			return p
		}(), change: func(n *capacity.Capacity) {
			n.HeldByDeleted = nil
		}, want: true},
		{name: "held by snapshots moved by little", prev: func() *capacity.Capacity {
			p := written(at)
			p.HeldBySnapshots = capacity.OptionalQuantity(gib)
			return p
		}(), change: func(n *capacity.Capacity) {
			n.HeldBySnapshots = capacity.OptionalQuantity(gib + 100*mib)
		}, want: false},
		{name: "free by allocation limit moved by more than one percent", prev: func() *capacity.Capacity {
			p := written(at)
			p.FreeByAllocationLimit = capacity.OptionalQuantity(50 * gib)
			return p
		}(), change: func(n *capacity.Capacity) {
			n.FreeByAllocationLimit = capacity.OptionalQuantity(45 * gib)
		}, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			next := written(now)
			if tc.prev != nil {
				next = tc.prev.DeepCopy()
				next.UpdatedAt = metav1.NewTime(now)
			}
			if tc.change != nil {
				tc.change(next)
			}
			at := now
			if !tc.now.IsZero() {
				at = tc.now
			}
			if got := capacity.NeedsWrite(tc.prev, next, at); got != tc.want {
				t.Fatalf("NeedsWrite = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNeedsWriteWithoutNext(t *testing.T) {
	now := time.Now()
	if capacity.NeedsWrite(nil, nil, now) {
		t.Error("nothing to write and nothing written, yet a write is needed")
	}
	if !capacity.NeedsWrite(written(now), nil, now) {
		t.Error("figures that are gone are kept")
	}
}

func TestOptionalFieldsAreLeftOut(t *testing.T) {
	at := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	out, err := json.Marshal(written(at))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"size":"100Gi","used":"40Gi","freeOnDisk":"60Gi","updatedAt":"2026-11-24T12:00:00Z"}`
	if string(out) != want {
		t.Fatalf("json = %s, want %s", out, want)
	}
}
