package objectstorage_test

import (
	"testing"

	"github.com/deckhouse/sds-common-lib/objectstorage"
)

// The order is part of the contract: the tokens end up in a condition message,
// and a set that reorders itself between reconciles rewrites the message (and the
// condition's lastTransitionTime with it) without anything having changed.
func TestRequestedFeaturesIsOrdered(t *testing.T) {
	cases := []struct {
		name       string
		quota      bool
		publicRead bool
		want       []string
	}{
		{name: "nothing requested"},
		{name: "quota only", quota: true, want: []string{objectstorage.FeatureQuota}},
		{name: "public read only", publicRead: true, want: []string{objectstorage.FeaturePublicRead}},
		{
			name: "both, quota first", quota: true, publicRead: true,
			want: []string{objectstorage.FeatureQuota, objectstorage.FeaturePublicRead},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := objectstorage.RequestedFeatures(objectstorage.Requested{Quota: c.quota, PublicRead: c.publicRead})
			if len(got) != len(c.want) {
				t.Fatalf("RequestedFeatures(%v, %v) = %v, want %v", c.quota, c.publicRead, got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("feature %d = %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

// The identifier has to be unique per requesting object and traceable back to it,
// which is what namespace-qualifying it buys: two namespaces may hold an access
// request of the same name, and they must not collide in the backend.
func TestAccessResourceNameIsNamespaceQualified(t *testing.T) {
	if got := objectstorage.AccessResourceName("team-a", "reader"); got != "team-a.reader" {
		t.Errorf("AccessResourceName = %q, want team-a.reader", got)
	}
	if a, b := objectstorage.AccessResourceName("team-a", "reader"), objectstorage.AccessResourceName("team-b", "reader"); a == b {
		t.Errorf("requests of the same name in different namespaces must not share an identifier (%q)", a)
	}
}

func TestEffectiveBucketNameFallsBackToTheObjectName(t *testing.T) {
	if got := objectstorage.EffectiveBucketName("", "app-data"); got != "app-data" {
		t.Errorf("EffectiveBucketName(unset) = %q, want the object name", got)
	}
	if got := objectstorage.EffectiveBucketName("chosen", "app-data"); got != "chosen" {
		t.Errorf("EffectiveBucketName(set) = %q, want the requested name", got)
	}
}
