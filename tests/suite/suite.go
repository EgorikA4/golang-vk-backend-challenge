package suite

import (
	"context"
	"testing"
	"time"

	"github.com/golang-vk-backend-challenge/subpub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Suite struct {
	t       *testing.T
	Subpub  *subpub.SubPub
	Cleanup func()
}

func NewSuite(t *testing.T) *Suite {
	t.Helper()
	t.Parallel()

	sp := subpub.NewSubPub()
	cleanup := func() {
		assert.NoError(t, sp.Close(context.Background()))
	}

	return &Suite{
		t:       t,
		Subpub:  sp,
		Cleanup: cleanup,
	}
}

func (st *Suite) MustPublish(topic string, msg any) {
	st.t.Helper()
	require.NoError(st.t, st.Subpub.Publish(topic, msg))
}

func (st *Suite) StartConsumer(sub *subpub.Subscription) {
	sub.WG.Add(1)
	go func() {
		defer sub.WG.Done()
		for msg := range sub.Queue {
			sub.Callback(msg)
		}
	}()
}

func (st *Suite) Subscribe(topic string, cb subpub.MessageHandler) *subpub.Subscription {
	st.t.Helper()

	sub, err := st.Subpub.Subscribe(topic, cb)
	require.NoError(st.t, err)
	return sub
}

func (st *Suite) CloseWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return st.Subpub.Close(ctx)
}
