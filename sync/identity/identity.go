// Package identity provides aa-sync peer identity, pairing, and mutual
// authentication.
//
// A peer's identity is an Ed25519 key pair; its node ID is the fingerprint of
// its public key. Loopback is not authentication: two nodes must be paired and
// must both prove possession of their private key before sync is allowed.
package identity

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"time"

	aasync "github.com/greadee/aa/sync"
)

// KeyPair is a node's Ed25519 identity.
type KeyPair struct {
	Private ed25519.PrivateKey
	Public  ed25519.PublicKey
}

// Generate derives a deterministic key pair from a seed. Seeds are for tests
// and local fixtures; production identities use a random seed.
func Generate(seed string) KeyPair {
	sum := sha256.Sum256([]byte("aa-sync-identity:" + seed))
	priv := ed25519.NewKeyFromSeed(sum[:])
	return KeyPair{Private: priv, Public: priv.Public().(ed25519.PublicKey)}
}

// Fingerprint returns the stable node ID for a public key.
func Fingerprint(pub ed25519.PublicKey) aasync.NodeID {
	sum := sha256.Sum256(pub)
	return aasync.NodeID(hex.EncodeToString(sum[:])[:16])
}

// ID returns the local node ID.
func (k KeyPair) ID() aasync.NodeID { return Fingerprint(k.Public) }

// Sign signs a message with the local key.
func (k KeyPair) Sign(msg []byte) []byte { return ed25519.Sign(k.Private, msg) }

// Node is a local or remote identity. A local node carries its private key.
type Node struct {
	ID     aasync.NodeID
	Public ed25519.PublicKey
	Name   string

	key *KeyPair
}

// NewNode returns a local node that can authenticate.
func NewNode(kp KeyPair, name string) Node {
	return Node{ID: kp.ID(), Public: kp.Public, Name: name, key: &kp}
}

// Remote returns a remote node reference that can be verified against.
func Remote(pub ed25519.PublicKey, name string) Node {
	return Node{ID: Fingerprint(pub), Public: pub, Name: name}
}

// Peer returns the shared peer representation.
func (n Node) Peer() aasync.Peer { return aasync.Peer{ID: n.ID, Name: n.Name} }

// HasPrivate reports whether the node can sign challenges.
func (n Node) HasPrivate() bool { return n.key != nil }

// Token is a single-use, time-bounded pairing token.
type Token struct {
	Value     string        `json:"value"`
	Node      aasync.NodeID `json:"node"`
	ExpiresAt time.Time     `json:"expiresAt"`
}

// Pairing is an established pairing between two nodes.
type Pairing struct {
	Local    aasync.Peer `json:"local"`
	Remote   aasync.Peer `json:"remote"`
	PairedAt time.Time   `json:"pairedAt"`
}

// PairingService issues and accepts pairing tokens for one local node.
// A token is derived from a shared secret and the remote node ID, and is
// accepted at most once.
type PairingService struct {
	local  Node
	secret []byte
	now    func() time.Time
	used   map[string]bool
}

// NewPairingService returns a pairing service for the local node.
func NewPairingService(local Node, secret string, now func() time.Time) *PairingService {
	if now == nil {
		now = time.Now
	}
	return &PairingService{local: local, secret: []byte(secret), now: now, used: map[string]bool{}}
}

// Issue returns a token that admits the remote node until ttl elapses.
func (s *PairingService) Issue(remote aasync.NodeID, ttl time.Duration) Token {
	expires := s.now().Add(ttl)
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(string(remote)))
	mac.Write([]byte(expires.UTC().Format(time.RFC3339Nano)))
	return Token{Value: hex.EncodeToString(mac.Sum(nil)), Node: remote, ExpiresAt: expires}
}

// Accept validates a token for the claiming remote node. A token that is
// expired, bound to another node, forged, or already used is rejected.
func (s *PairingService) Accept(token Token, claimer aasync.NodeID) error {
	if token.Value == "" || token.Node == "" || claimer == "" {
		return aasync.ErrInvalid
	}
	if token.Node != claimer {
		return aasync.ErrUnauthorized
	}
	if !token.ExpiresAt.After(s.now()) {
		return aasync.ErrStale
	}
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(string(token.Node)))
	mac.Write([]byte(token.ExpiresAt.UTC().Format(time.RFC3339Nano)))
	want := hex.EncodeToString(mac.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(want), []byte(token.Value)) != 1 {
		return aasync.ErrUnauthorized
	}
	if s.used[token.Value] {
		return aasync.ErrStale
	}
	s.used[token.Value] = true
	return nil
}

// challengeFor binds a nonce to both node IDs so a signature cannot be
// replayed on a different channel.
func challengeFor(a, b aasync.NodeID, nonce []byte) []byte {
	out := make([]byte, 0, 24+len(a)+len(b)+len(nonce))
	out = append(out, []byte("aa-sync-auth:")...)
	out = append(out, []byte(a)...)
	out = append(out, ':')
	out = append(out, []byte(b)...)
	out = append(out, ':')
	return append(out, nonce...)
}

// Respond signs the peer's nonce, bound to both node IDs.
func Respond(local Node, peer aasync.NodeID, nonce []byte) ([]byte, error) {
	if !local.HasPrivate() {
		return nil, aasync.ErrUnauthorized
	}
	return local.key.Sign(challengeFor(peer, local.ID, nonce)), nil
}

// Verify checks that sig is the peer's signature over the local node's nonce.
func Verify(peer Node, local aasync.NodeID, nonce, sig []byte) error {
	if len(sig) != ed25519.SignatureSize {
		return aasync.ErrUnauthorized
	}
	if !ed25519.Verify(peer.Public, challengeFor(local, peer.ID, nonce), sig) {
		return aasync.ErrUnauthorized
	}
	return nil
}

// Nonce returns a fresh random challenge nonce.
func Nonce() ([]byte, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// MutualAuth runs a full challenge-response between two local nodes and
// returns the established pairing. It is deterministic when nonces are
// supplied.
func MutualAuth(a, b Node, now time.Time) (Pairing, error) {
	na, err := Nonce()
	if err != nil {
		return Pairing{}, err
	}
	nb, err := Nonce()
	if err != nil {
		return Pairing{}, err
	}
	return MutualAuthWithNonces(a, b, na, nb, now)
}

// MutualAuthWithNonces is MutualAuth with explicit nonces for deterministic
// tests and replays.
func MutualAuthWithNonces(a, b Node, nonceA, nonceB []byte, now time.Time) (Pairing, error) {
	if !a.HasPrivate() || !b.HasPrivate() {
		return Pairing{}, aasync.ErrUnauthorized
	}
	// a proves itself over b's nonce; b proves itself over a's nonce.
	sigAByA, err := Respond(a, b.ID, nonceB)
	if err != nil {
		return Pairing{}, err
	}
	if err := Verify(a, b.ID, nonceB, sigAByA); err != nil {
		return Pairing{}, err
	}
	sigBByB, err := Respond(b, a.ID, nonceA)
	if err != nil {
		return Pairing{}, err
	}
	if err := Verify(b, a.ID, nonceA, sigBByB); err != nil {
		return Pairing{}, err
	}
	return Pairing{Local: a.Peer(), Remote: b.Peer(), PairedAt: now}, nil
}
