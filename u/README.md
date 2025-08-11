## Package `u`

Small, focused helpers for everyday tasks; the package name "u" stands for "utilities". Use these to avoid rewriting tiny helpers across services.

### Import

```go
import (
    "github.com/deckhouse/sds-common-lib/u"
)
```

or even

```go
import (
    . "github.com/deckhouse/sds-common-lib/u"
)
```

### Quick examples

Pointers:

```go
v := 10
p := u.Ptr(v) // *int
```

Errors and logging:

```go
defer u.RecoverPanicToErr(&err)
_ = u.LogError(log, err)
```

Iterators and slices (using `iter`):

```go
keys := u.IterToKeys(mySeq)
mapped := u.IterMap(mySeq, func(x int) string { return strconv.Itoa(x) })

found := u.SliceFind(items, func(v *Item) bool { return v.ID == id })
for ptr := range u.SliceFilter(items, func(v *Item) bool { return v.Ready }) {
    // use ptr
}
```

Forever-running goroutines:

```go
ctx, cancel := context.WithCancelCause(context.Background())
u.GoForever("worker", cancel, log, func() error {
    // run until error
    return someErr
})
```

Maps:

```go
var m map[string]int
u.MapEnsureAndSet(&m, "a", 1)
```

