package subpub

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

const buffSize = 5

type MessageHandler func(msg any)

type Subscription struct {
	Queue    chan any
	Callback MessageHandler
	WG       sync.WaitGroup
	IsClosed atomic.Bool
}

func NewSubscription(buffSize int, cb MessageHandler) *Subscription {
	return &Subscription{
		Queue:    make(chan any, buffSize),
		Callback: cb,
	}
}

func (s *Subscription) Unsubscribe() {
	if s.IsClosed.Load() {
		return
	}
	s.IsClosed.Store(true)

	close(s.Queue)
	s.WG.Wait()
}

type SubPub struct {
	Topics map[string][]*Subscription
	mu     sync.Mutex
}

func NewSubPub() *SubPub {
	return &SubPub{
		Topics: make(map[string][]*Subscription),
	}
}

func (sp *SubPub) Subscribe(subject string, cb MessageHandler) (*Subscription, error) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	subscription := NewSubscription(buffSize, cb)
	sp.Topics[subject] = append(sp.Topics[subject], subscription)
	return subscription, nil
}

func (sp *SubPub) Publish(subject string, msg any) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	subscriptions, ok := sp.Topics[subject]
	if !ok {
		// TODO: вынести в константы сообщение об ошибке
		return fmt.Errorf("subject: %s does not exist", subject)
	}

	for _, subscription := range subscriptions {
		if !subscription.IsClosed.Load() {
			select {
			case subscription.Queue <- msg:
			default:
				// Skip if buffer is full (slow subscriber)
			}
		}
	}
	return nil
}

func (sp *SubPub) Close(ctx context.Context) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	done := make(chan struct{})
	go func() {
		for _, subscriptions := range sp.Topics {
			for _, subscription := range subscriptions {
				subscription.Unsubscribe()
			}
		}
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
