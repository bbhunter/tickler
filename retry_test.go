package tickler

import (
	"testing"
	"time"
)

func Test1(t *testing.T) {
	b := &backoff{
		Min:    100 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 2,
	}

	if !assert(b.Duration(), time.Millisecond*100) {
		t.Fatalf("step 1 failed")
	}

	if !assert(b.Duration(), time.Millisecond*200) {
		t.Fatalf("step 2 failed")
	}
	if !assert(b.Duration(), time.Millisecond*400) {
		t.Fatalf("step 3 failed")
	}

	if !assert(b.Duration(), time.Millisecond*800) {
		t.Fatalf("step 4 failed")
	}
}

func assert[T comparable](left, right T) bool {
	return left == right
}
