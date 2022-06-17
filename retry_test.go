package tickler

import (
	"github.com/goodjobtech/assert"
	"testing"
	"time"
)

func TestBackoff_Next(t *testing.T) {
	b := &backoff{
		Min:    100 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 2,
	}

	assert.Equal(t, time.Millisecond*200, b.Next(1))
}

func TestBackoff_Duration(t *testing.T) {
	b := &backoff{
		Min:    100 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 2,
	}

	assert.Equal(t, time.Millisecond*100, b.Duration())
	assert.Equal(t, time.Millisecond*200, b.Duration())
	assert.Equal(t, time.Millisecond*400, b.Duration())
	assert.Equal(t, time.Millisecond*800, b.Duration())
}
