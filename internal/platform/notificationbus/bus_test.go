package notificationbus

import (
	"context"
	"testing"
)

func TestBusDeliversOnlyToMatchingUser(t *testing.T) {
	bus := New()
	userEvents, unsubscribeUser := bus.Subscribe(7)
	defer unsubscribeUser()
	otherEvents, unsubscribeOther := bus.Subscribe(8)
	defer unsubscribeOther()

	if err := bus.NotifyUser(context.Background(), 7, "mail.notification", map[string]string{"id": "1"}); err != nil {
		t.Fatalf("notify user: %v", err)
	}

	select {
	case event := <-userEvents:
		if event.Channel != "mail.notification" {
			t.Fatalf("unexpected channel %q", event.Channel)
		}
	case <-otherEvents:
		t.Fatal("event was delivered to the wrong user")
	default:
		t.Fatal("expected matching user to receive an event")
	}

	select {
	case <-otherEvents:
		t.Fatal("event was delivered to another user")
	default:
	}
}

func TestBusUnsubscribeClosesSubscription(t *testing.T) {
	bus := New()
	events, unsubscribe := bus.Subscribe(7)
	unsubscribe()

	select {
	case _, ok := <-events:
		if ok {
			t.Fatal("expected unsubscribed channel to be closed")
		}
	default:
		t.Fatal("expected unsubscribe to close the channel")
	}
}
