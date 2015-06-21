package couchbasearray

import (
	"testing"
	"time"
)

func TestTTL(t *testing.T) {
	old := time.Now().UnixNano()
	new := time.Now().Add(time.Duration(1) * time.Second).UnixNano()
	if old > new {
		t.Fatal("Unexpected compare")
	}
}
