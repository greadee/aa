package identity

import (
	"errors"
	"testing"
	"time"

	aasync "github.com/greadee/aa/sync"
)

func testClock(start time.Time) func() time.Time {
	cur := start
	return func() time.Time {
		cur = cur.Add(time.Second)
		return cur
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	a := Generate("node-a")
	b := Generate("node-a")
	c := Generate("node-b")
	if a.ID() != b.ID() {
		t.Fatal("same seed produced different identities")
	}
	if a.ID() == c.ID() {
		t.Fatal("different seeds produced the same identity")
	}
	if len(a.ID()) != 16 {
		t.Fatalf("fingerprint length = %d, want 16", len(a.ID()))
	}
}

func TestPairingTokenLifecycle(t *testing.T) {
	kp := Generate("local")
	local := NewNode(kp, "local")
	remote := Remote(Generate("remote").Public, "remote")
	svc := NewPairingService(local, "shared-secret", testClock(time.Unix(1000, 0).UTC()))

	token := svc.Issue(remote.ID, 5*time.Minute)
	if err := svc.Accept(token, remote.ID); err != nil {
		t.Fatalf("first accept: %v", err)
	}
	if err := svc.Accept(token, remote.ID); !errors.Is(err, aasync.ErrStale) {
		t.Fatalf("reused token err = %v, want ErrStale", err)
	}
}

func TestPairingTokenRejections(t *testing.T) {
	local := NewNode(Generate("local-2"), "local")
	remote := Remote(Generate("remote-2").Public, "remote")
	other := Remote(Generate("other-2").Public, "other")
	now := time.Unix(2000, 0).UTC()
	svc := NewPairingService(local, "secret", func() time.Time { return now })

	token := svc.Issue(remote.ID, time.Minute)
	if err := svc.Accept(token, other.ID); !errors.Is(err, aasync.ErrUnauthorized) {
		t.Fatalf("wrong node err = %v, want ErrUnauthorized", err)
	}
	forged := token
	forged.Value = "deadbeef"
	if err := svc.Accept(forged, remote.ID); !errors.Is(err, aasync.ErrUnauthorized) {
		t.Fatalf("forged token err = %v, want ErrUnauthorized", err)
	}

	expired := svc.Issue(remote.ID, time.Minute)
	later := NewPairingService(local, "secret", func() time.Time { return now.Add(2 * time.Minute) })
	if err := later.Accept(expired, remote.ID); !errors.Is(err, aasync.ErrStale) {
		t.Fatalf("expired token err = %v, want ErrStale", err)
	}
}

func TestMutualAuthSucceeds(t *testing.T) {
	a := NewNode(Generate("a"), "a")
	b := NewNode(Generate("b"), "b")
	na := []byte("nonce-a-nonce-a-nonce-a-nonce-a!")
	nb := []byte("nonce-b-nonce-b-nonce-b-nonce-b!")
	pairing, err := MutualAuthWithNonces(a, b, na, nb, time.Unix(3000, 0).UTC())
	if err != nil {
		t.Fatalf("mutual auth: %v", err)
	}
	if pairing.Local.ID != a.ID || pairing.Remote.ID != b.ID {
		t.Fatalf("unexpected pairing: %+v", pairing)
	}
}

func TestMutualAuthRejectsImpostor(t *testing.T) {
	a := NewNode(Generate("a3"), "a")
	b := NewNode(Generate("b3"), "b")
	impostor := NewNode(Generate("c3"), "c")
	nb := []byte("nonce-b-nonce-b-nonce-b-nonce-b!")

	sig, err := Respond(impostor, b.ID, nb)
	if err != nil {
		t.Fatalf("respond: %v", err)
	}
	if err := Verify(a, b.ID, nb, sig); !errors.Is(err, aasync.ErrUnauthorized) {
		t.Fatalf("impostor verify err = %v, want ErrUnauthorized", err)
	}
	if _, err := MutualAuth(a, b, time.Now()); err != nil {
		t.Fatalf("mutual auth: %v", err)
	}
}

func TestRespondRequiresPrivateKey(t *testing.T) {
	local := NewNode(Generate("pub-only"), "local")
	remote := Remote(Generate("remote-4").Public, "remote")
	if _, err := Respond(remote, local.ID, []byte("nonce")); !errors.Is(err, aasync.ErrUnauthorized) {
		t.Fatalf("respond err = %v, want ErrUnauthorized", err)
	}
}
