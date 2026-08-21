// Package objectstorage is the vocabulary the S3 object-storage modules share
// with their backend drivers: what a driver reports back about a bucket and an
// access key, how a bucket and a key are named in the backend, and which
// optional bucket features a backend may or may not be able to enforce.
//
// Two modules provision buckets on their own backends — sds-object (SeaweedFS,
// Ceph RGW) for tenants, sds-object-system (Garage) for the platform itself —
// and the split between them is deliberate: their backends have different
// licences and different consumers. What is not deliberate is each of them
// carrying its own copy of these types, which is how the copies drifted (one grew
// a report of unenforceable features, the other did not).
//
// What is NOT here is the Driver interface itself. Each module's driver is handed
// its own custom resources, and an interface general enough for both would be
// parameterised by every one of those types — coupling both APIs to this package
// to save four method signatures. The interface stays in the modules; the values
// that cross it live here.
//
// The observed state of a data plane is likewise not here: one module describes
// its store with a custom resource and reports typed status objects, the other
// has no store resource at all. There is nothing shared to name.
package objectstorage

import "fmt"

// BucketState is the observed state a driver reports for one bucket. Credentials
// are not part of it: access keys are issued per access request, and reported
// through [AccessState].
type BucketState struct {
	// Ready is true once the bucket exists in the backend.
	Ready bool

	// Message is a human-readable explanation of the current state, used as the
	// condition message (especially while not Ready).
	Message string

	// BucketName is the effective bucket name created in the backend.
	BucketName string

	// UnsupportedFeatures lists the features that were requested on the bucket
	// but that this backend cannot enforce (a quota on a backend with no quota
	// support, for one). The reconciler surfaces them so that a requested feature
	// never silently no-ops. An empty slice means everything requested was
	// applied.
	//
	// The tokens are the Feature* constants in this package: they end up in a
	// condition message an operator reads, so they name the spec field that was
	// not honoured rather than something internal to the driver.
	UnsupportedFeatures []string
}

// AccessState is the observed state a driver reports for one access key.
//
// SecretAccessKey is populated only when a fresh key was just issued; when it is
// empty the access already had a live key and the existing credentials Secret is
// preserved. That is not an optimisation but the only correct behaviour: no S3
// backend in use here can recover a secret key after creation, so a reconcile
// that "refreshed" the Secret from an empty value would lock the consumer out.
type AccessState struct {
	// Ready is true once a key scoped to the bucket is provisioned.
	Ready bool

	// Message is a human-readable explanation of the current state.
	Message string

	// AccessKeyID is the public access key id.
	AccessKeyID string

	// SecretAccessKey is the secret key; populated only when a fresh key was
	// just issued — see the type comment.
	SecretAccessKey string
}

// Bucket spec features a backend may or may not be able to enforce. They are the
// tokens in [BucketState.UnsupportedFeatures], so they name the field the user
// wrote.
const (
	FeatureQuota      = "spec.quota"
	FeaturePublicRead = "spec.accessPolicy=PublicRead"
)

// Requested says which optional features a bucket's spec asks for. It is a struct
// rather than a pair of bool arguments so that a call site cannot silently swap
// them: two adjacent booleans of the same type read identically to the compiler
// and differently to the backend.
type Requested struct {
	// Quota is true when the spec asks for any usage limit at all.
	Quota bool

	// PublicRead is true when the spec asks for anonymous read access.
	PublicRead bool
}

// RequestedFeatures returns the optional, backend-dependent features a bucket
// asks for, in a fixed order. A driver compares it against what it can enforce to
// populate [BucketState.UnsupportedFeatures].
//
// It takes the answers rather than a spec, because the specs differ between the
// modules: quota is a struct in one and absent in the other, and the access
// policy is an enum whose values are declared per module. Deciding "did the user
// ask for a quota" is the caller's job; deciding what the answer is called is
// this package's.
//
// The order is part of the contract: the tokens end up in a condition message, and
// a set that reorders itself between reconciles rewrites the message — and the
// condition's lastTransitionTime with it — without anything having changed.
func RequestedFeatures(r Requested) []string {
	var out []string
	if r.Quota {
		out = append(out, FeatureQuota)
	}
	if r.PublicRead {
		out = append(out, FeaturePublicRead)
	}
	return out
}

// AccessResourceName is the backend-facing identifier for an access request — the
// key display name, IAM identity or RGW user, depending on the backend. It is
// derived from the requesting object's namespace and name, which makes it unique
// per request and traceable back to it from the backend side.
func AccessResourceName(namespace, name string) string {
	return fmt.Sprintf("%s.%s", namespace, name)
}

// EffectiveBucketName is the S3 bucket name a request maps to: the name it asked
// for, or the object's own name when it asked for none.
func EffectiveBucketName(requested, fallback string) string {
	if requested != "" {
		return requested
	}
	return fallback
}
