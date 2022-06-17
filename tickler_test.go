package tickler

import (
	"context"
	"github.com/goodjobtech/assert"
	"testing"
)

func TestTickler_New(t *testing.T) {
	tl := New()
	assert.NotNil(t, tl.queue)
}

func TestTickler_Enqueue(t *testing.T) {
	tl := New()
	tl.Enqueue(Request{
		Job: func() error {
			return nil
		},
		Name: "test",
	})

	assert.NotEqual(t, 0, tl.queue.Len())
}

func TestTickler_GetQueueLength(t *testing.T) {
	tl := New()
	tl.Enqueue(Request{
		Job: func() error {
			return nil
		},
		Name: "test",
	})

	assert.Equal(t, 1, tl.GetQueueLength())
}

func TestTickler_GetCurrentJobs(t *testing.T) {
	tl := New()
	tl.Enqueue(Request{
		Job: func() error {
			return nil
		},
		Name: "test",
	})

	assert.Equal(t, true, tl.GetCurrentJobs()["test"])
}

func TestTickler_GetContext(t *testing.T) {
	tl := New()
	ctx := tl.GetContext()

	assert.NotNil(t, ctx)
}

func TestTickler_SetContext(t *testing.T) {
	tl := New()

	ctx := context.WithValue(context.Background(), "name", "test")
	tl.SetContext(ctx)

	expected := ctx.Value("name").(string)
	actual := tl.GetContext().Value("name").(string)

	assert.Equal(t, expected, actual)
}

func TestTickler_Limit(t *testing.T) {
	tl := New()

	limit := 10
	tl.Limit(limit)

	assert.Equal(t, int64(limit), tl.options.Limit)
}
