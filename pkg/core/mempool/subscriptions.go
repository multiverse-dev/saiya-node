package mempool

import "github.com/multiverse-dev/saiya/pkg/core/mempoolevent"

// RunSubscriptions runs subscriptions goroutine if mempool subscriptions are enabled.
// You should manually free the resources by calling StopSubscriptions on mempool shutdown.
func (mp *Pool) RunSubscriptions() {
	if !mp.subscriptionsEnabled {
		panic("subscriptions are disabled")
	}
	if !mp.subscriptionsOn.Load() {
		mp.subscriptionsOn.Store(true)
		go mp.notificationDispatcher()
	}
}

// StopSubscriptions stops mempool events loop.
func (mp *Pool) StopSubscriptions() {
	if !mp.subscriptionsEnabled {
		panic("subscriptions are disabled")
	}
	if mp.subscriptionsOn.Load() {
		mp.subscriptionsOn.Store(false)
		close(mp.stopCh)
	}
}

// SubscribeForTransactions adds given channel to new mempool event broadcasting, so when
// there is a new transactions added to mempool or an existing transaction removed from
// mempool you'll receive it via this channel.
func (mp *Pool) SubscribeForTransactions(ch chan<- mempoolevent.Event) {
	if mp.subscriptionsOn.Load() {
		mp.subCh <- ch
	}
}

// UnsubscribeFromTransactions unsubscribes given channel from new mempool notifications,
// you can close it afterwards. Passing non-subscribed channel is a no-op.
func (mp *Pool) UnsubscribeFromTransactions(ch chan<- mempoolevent.Event) {
	if mp.subscriptionsOn.Load() {
		mp.unsubCh <- ch
	}
}

// notificationDispatcher manages subscription to events and broadcasts new events.
func (mp *Pool) notificationDispatcher() {
	var (
		// These are just sets of subscribers, though modelled as maps
		// for ease of management (not a lot of subscriptions is really
		// expected, but maps are convenient for adding/deleting elements).
		txFeed = make(map[chan<- mempoolevent.Event]bool)
	)
	for {
		select {
		case <-mp.stopCh:
			return
		case sub := <-mp.subCh:
			txFeed[sub] = true
		case unsub := <-mp.unsubCh:
			delete(txFeed, unsub)
		case event := <-mp.events:
			for ch := range txFeed {
				ch <- event
			}
		}
	}
}
