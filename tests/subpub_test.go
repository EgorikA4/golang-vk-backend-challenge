package tests

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/golang-vk-backend-challenge/subpub"
	"github.com/golang-vk-backend-challenge/tests/suite"
	"github.com/stretchr/testify/assert"
)

const (
	defaultBuffSize = 5
	defaultTopic    = "default-topic"
	testTimeout     = 10 * time.Millisecond
)

func TestNewSubscription(t *testing.T) {
	cb := func(msg any) {}
	subscription := subpub.NewSubscription(defaultBuffSize, cb)

	assert.NotNil(t, subscription.Queue)
	assert.Equal(t, defaultBuffSize, cap(subscription.Queue))
	assert.Equal(t, reflect.ValueOf(cb).Pointer(), reflect.ValueOf(subscription.Callback).Pointer())
}

func TestSubscribe(t *testing.T) {
	st := suite.NewSuite(t)
	defer st.Cleanup()

	sub := st.Subscribe(defaultTopic, func(msg any) {})

	assert.Len(t, st.Subpub.Topics, 1)
	assert.Len(t, st.Subpub.Topics[defaultTopic], 1)
	assert.Equal(t, sub, st.Subpub.Topics[defaultTopic][0])
}

func TestSubscribe_DuplicateTopic(t *testing.T) {
	st := suite.NewSuite(t)
	defer st.Cleanup()

	cb1 := func(msg any) {}
	cb2 := func(msg any) {}

	sub1 := st.Subscribe(defaultTopic, cb1)
	sub2 := st.Subscribe(defaultTopic, cb2)

	assert.Len(t, st.Subpub.Topics[defaultTopic], 2)
	assert.Equal(t, sub1, st.Subpub.Topics[defaultTopic][0])
	assert.Equal(t, sub2, st.Subpub.Topics[defaultTopic][1])
}

func TestPublish_MessageDelivery(t *testing.T) {
	st := suite.NewSuite(t)
	defer st.Cleanup()

	msg := "test message"
	received := make(chan any)

	sub := st.Subscribe(defaultTopic, func(msg any) {
		received <- msg
	})

	st.StartConsumer(sub)
	st.MustPublish(defaultTopic, msg)

	select {
	case receivedMsg := <-received:
		assert.Equal(t, msg, receivedMsg)
	case <-time.After(testTimeout):
		assert.Fail(t, "The message was not delivered during the timeout.")
	}
}

func TestPublish_BufferFull(t *testing.T) {
	st := suite.NewSuite(t)
	defer st.Cleanup()

	const bufferSize = 2
	sub := st.Subscribe(defaultTopic, func(msg any) {})
	sub.Queue = make(chan any, bufferSize)

	for i := 0; i < bufferSize; i++ {
		st.MustPublish(defaultTopic, i)
	}

	st.MustPublish(defaultTopic, "full")
}

func TestPublish_UnknownTopic(t *testing.T) {
	st := suite.NewSuite(t)
	defer st.Cleanup()

	err := st.Subpub.Publish("unknown", "msg")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subject: unknown does not exist")
}

func TestUnsubscribe(t *testing.T) {
	st := suite.NewSuite(t)
	defer st.Cleanup()

	sub := st.Subscribe(defaultTopic, func(msg any) {})

	st.StartConsumer(sub)
	sub.Unsubscribe()

	assert.True(t, sub.IsClosed.Load())
}

func TestClose(t *testing.T) {
	st := suite.NewSuite(t)

	const topicsCount = 3
	subs := make([]*subpub.Subscription, 0)
	for i := 0; i < topicsCount; i++ {
		topic := fmt.Sprintf("topic-%d", i)
		subs = append(subs, st.Subscribe(topic, func(msg any) {}))
	}

	for _, sub := range subs {
		st.StartConsumer(sub)
	}

	st.Cleanup()
	for _, sub := range subs {
		assert.Equal(t, sub.IsClosed.Load(), true)
	}
}

func TestCloseWithContextTimeout(t *testing.T) {
	st := suite.NewSuite(t)

	sub := st.Subscribe(defaultTopic, func(msg any) {
		select {}
	})

	st.StartConsumer(sub)
	st.MustPublish(defaultTopic, "testMSG")

	err := st.CloseWithTimeout(10 * time.Millisecond)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))

	sub.Unsubscribe()
}

func TestSubscriptionLifecycle(t *testing.T) {
	st := suite.NewSuite(t)

	msg := "lifecycle test"
	received := make(chan struct{})

	sub := st.Subscribe(defaultTopic, func(msg any) {
		close(received)
	})

	st.StartConsumer(sub)
	st.MustPublish(defaultTopic, msg)

	select {
	case <-received:
	case <-time.After(testTimeout):
		assert.Fail(t, "Message not delivered before closing")
	}

	sub.Unsubscribe()
	st.MustPublish(defaultTopic, "after close")

	st.Cleanup()
	assert.True(t, sub.IsClosed.Load())
}
