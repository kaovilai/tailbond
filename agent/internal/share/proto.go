// Package share is Pair & Share: a device on the LAN (a phone, a laptop
// with an LTE dongle) lends its cellular uplink to a hub (the travel router
// or a Linux client), which bonds it as one more mqvpn path.
//
// The sharer runs a small UDP relay. The hub opens a session with an HMAC
// challenge, then each mqvpn datagram travels as a DATA frame hub → sharer
// → (cellular) → home server and back. The relay only ever talks to its own
// home server, so it can't be used to reach anything else.
//
// Keys (docs/architecture.md "Pair & Share"): every paired device has its
// own share key, derived by the server from a secret only it holds, the
// device's name and its pairing (ClientKey). A sharer holds its own. A hub
// gets from the server, for each other device, a pass: PassKey(that
// device's key, hub name, expiry). The sharer derives the same pass from
// its own key and the hub's name and expiry in HELLO, so it checks the hub
// offline; a hub can't claim another hub's name, and a revoked one gets no
// new passes, so its access ends when its last pass expires.
//
// Wire format (all integers big-endian):
//
//	HELLO      01 ver hub_nonce[16] exp[8] idlen id
//	CHALLENGE  02 ver hub_nonce[16] sharer_nonce[16] idlen id lablen label
//	AUTH       03 ver hub_nonce[16] sharer_nonce[16] mac[32]
//	WELCOME    04 ver hub_nonce[16] session[8] cap_left[8] mac[32]
//	REFUSE     05 ver hub_nonce[16] reason mac[32]
//	DATA       10 session[8] payload
//	KEEPALIVE  11 session[8] seq[4]
//	KA_ACK     12 session[8] seq[4] up[8] down[8] cap_left[8] battery flags
//	BYE        13 session[8] reason
//
// mac = HMAC-SHA256(pass, label ‖ hub_nonce ‖ sharer_nonce ‖ …), see the
// mac* functions; exp is the pass's expiry (unix seconds). cap_left is in
// bytes, ^0 = no cap.
package share

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
)

const (
	Version = 1
	// DefaultPort is the sharer's UDP port.
	DefaultPort = 47077
	// Service is the mDNS service type sharers advertise.
	Service = "_tailbond-share._udp"

	TypeHello     = 0x01
	TypeChallenge = 0x02
	TypeAuth      = 0x03
	TypeWelcome   = 0x04
	TypeRefuse    = 0x05
	TypeData      = 0x10
	TypeKeepalive = 0x11
	TypeKAAck     = 0x12
	TypeBye       = 0x13

	NonceLen   = 16
	SessionLen = 8
	MACLen     = 32
	// DataHeader is a DATA frame's overhead.
	DataHeader = 1 + SessionLen
	// NoCap is cap_left without a cap.
	NoCap = ^uint64(0)
	// MaxIDLen bounds hub/sharer ids and labels on the wire.
	MaxIDLen = 64
)

// Refuse/BYE reasons.
const (
	ReasonNone       = 0
	ReasonNotAllowed = 1 // the hub isn't in the sharer's allowed list
	ReasonCap        = 2 // the sharer's data cap is used up
	ReasonBattery    = 3 // battery below the sharer's limit, not charging
	ReasonPaused     = 4 // sharing turned off
	ReasonNoCellular = 5 // the sharer has no cellular network right now
	ReasonReplaced   = 6 // a newer session from the same hub
	ReasonIdle       = 7 // no keepalive
	ReasonExpired    = 8 // the hub's pass expired
)

// ReasonText is a reason in plain words.
func ReasonText(r byte) string {
	switch r {
	case ReasonNotAllowed:
		return "this device isn't allowed to use it"
	case ReasonCap:
		return "its monthly data limit is reached"
	case ReasonBattery:
		return "its battery is low"
	case ReasonPaused:
		return "sharing was turned off or paused on the phone"
	case ReasonNoCellular:
		return "it has no mobile data right now"
	case ReasonReplaced:
		return "replaced by a newer connection"
	case ReasonIdle:
		return "no answer"
	case ReasonExpired:
		return "the pass from the home server expired"
	}
	return "stopped"
}

var errShort = errors.New("share: short frame")

// Hello opens a handshake.
type Hello struct {
	HubNonce [NonceLen]byte
	Exp      uint64 // the hub's pass expiry, unix seconds
	HubID    string
}

// Challenge answers a Hello.
type Challenge struct {
	HubNonce, SharerNonce [NonceLen]byte
	SharerID, Label       string
}

// Auth proves the hub holds a pass for this sharer.
type Auth struct {
	HubNonce, SharerNonce [NonceLen]byte
	MAC                   [MACLen]byte
}

// Welcome opens the session and proves the sharer holds its key.
type Welcome struct {
	HubNonce [NonceLen]byte
	Session  [SessionLen]byte
	CapLeft  uint64
	MAC      [MACLen]byte
}

// Refuse turns a hub down after a valid Auth.
type Refuse struct {
	HubNonce [NonceLen]byte
	Reason   byte
	MAC      [MACLen]byte
}

// KAAck answers a keepalive with the sharer's counters for this session.
type KAAck struct {
	Session        [SessionLen]byte
	Seq            uint32
	Up, Down       uint64 // payload bytes hub→server, server→hub
	CapLeft        uint64
	Battery, Flags byte // battery percent (255 = unknown); Flags bit 0 = charging
}

// FlagCharging is KAAck.Flags bit 0.
const FlagCharging = 1

func putStr(b []byte, s string) []byte {
	if len(s) > MaxIDLen {
		s = s[:MaxIDLen]
	}
	b = append(b, byte(len(s)))
	return append(b, s...)
}

func getStr(b []byte) (string, []byte, error) {
	if len(b) < 1 || len(b) < 1+int(b[0]) {
		return "", nil, errShort
	}
	n := int(b[0])
	return string(b[1 : 1+n]), b[1+n:], nil
}

func head(t byte, b []byte) ([]byte, error) {
	if len(b) < 2 || b[0] != t {
		return nil, errShort
	}
	if b[1] != Version {
		return nil, errors.New("share: unsupported version")
	}
	return b[2:], nil
}

func (m *Hello) Marshal() []byte {
	b := append([]byte{TypeHello, Version}, m.HubNonce[:]...)
	b = binary.BigEndian.AppendUint64(b, m.Exp)
	return putStr(b, m.HubID)
}

func (m *Hello) Unmarshal(b []byte) error {
	r, err := head(TypeHello, b)
	if err != nil || len(r) < NonceLen+8 {
		return errShort
	}
	copy(m.HubNonce[:], r)
	m.Exp = binary.BigEndian.Uint64(r[NonceLen:])
	m.HubID, _, err = getStr(r[NonceLen+8:])
	return err
}

func (m *Challenge) Marshal() []byte {
	b := append([]byte{TypeChallenge, Version}, m.HubNonce[:]...)
	b = append(b, m.SharerNonce[:]...)
	b = putStr(b, m.SharerID)
	return putStr(b, m.Label)
}

func (m *Challenge) Unmarshal(b []byte) error {
	r, err := head(TypeChallenge, b)
	if err != nil || len(r) < 2*NonceLen {
		return errShort
	}
	copy(m.HubNonce[:], r)
	copy(m.SharerNonce[:], r[NonceLen:])
	r = r[2*NonceLen:]
	if m.SharerID, r, err = getStr(r); err != nil {
		return err
	}
	m.Label, _, err = getStr(r)
	return err
}

func (m *Auth) Marshal() []byte {
	b := append([]byte{TypeAuth, Version}, m.HubNonce[:]...)
	b = append(b, m.SharerNonce[:]...)
	return append(b, m.MAC[:]...)
}

func (m *Auth) Unmarshal(b []byte) error {
	r, err := head(TypeAuth, b)
	if err != nil || len(r) < 2*NonceLen+MACLen {
		return errShort
	}
	copy(m.HubNonce[:], r)
	copy(m.SharerNonce[:], r[NonceLen:])
	copy(m.MAC[:], r[2*NonceLen:])
	return nil
}

func (m *Welcome) Marshal() []byte {
	b := append([]byte{TypeWelcome, Version}, m.HubNonce[:]...)
	b = append(b, m.Session[:]...)
	b = binary.BigEndian.AppendUint64(b, m.CapLeft)
	return append(b, m.MAC[:]...)
}

func (m *Welcome) Unmarshal(b []byte) error {
	r, err := head(TypeWelcome, b)
	if err != nil || len(r) < NonceLen+SessionLen+8+MACLen {
		return errShort
	}
	copy(m.HubNonce[:], r)
	copy(m.Session[:], r[NonceLen:])
	m.CapLeft = binary.BigEndian.Uint64(r[NonceLen+SessionLen:])
	copy(m.MAC[:], r[NonceLen+SessionLen+8:])
	return nil
}

func (m *Refuse) Marshal() []byte {
	b := append([]byte{TypeRefuse, Version}, m.HubNonce[:]...)
	b = append(b, m.Reason)
	return append(b, m.MAC[:]...)
}

func (m *Refuse) Unmarshal(b []byte) error {
	r, err := head(TypeRefuse, b)
	if err != nil || len(r) < NonceLen+1+MACLen {
		return errShort
	}
	copy(m.HubNonce[:], r)
	m.Reason = r[NonceLen]
	copy(m.MAC[:], r[NonceLen+1:])
	return nil
}

func (m *KAAck) Marshal() []byte {
	b := append([]byte{TypeKAAck}, m.Session[:]...)
	b = binary.BigEndian.AppendUint32(b, m.Seq)
	b = binary.BigEndian.AppendUint64(b, m.Up)
	b = binary.BigEndian.AppendUint64(b, m.Down)
	b = binary.BigEndian.AppendUint64(b, m.CapLeft)
	return append(b, m.Battery, m.Flags)
}

func (m *KAAck) Unmarshal(b []byte) error {
	if len(b) < 1+SessionLen+4+24+2 || b[0] != TypeKAAck {
		return errShort
	}
	copy(m.Session[:], b[1:])
	r := b[1+SessionLen:]
	m.Seq = binary.BigEndian.Uint32(r)
	m.Up = binary.BigEndian.Uint64(r[4:])
	m.Down = binary.BigEndian.Uint64(r[12:])
	m.CapLeft = binary.BigEndian.Uint64(r[20:])
	m.Battery, m.Flags = r[28], r[29]
	return nil
}

// Data frames a payload for a session (into dst, which may be nil).
func Data(dst []byte, s [SessionLen]byte, payload []byte) []byte {
	dst = append(dst[:0], TypeData)
	dst = append(dst, s[:]...)
	return append(dst, payload...)
}

// Keepalive frames a keepalive.
func Keepalive(s [SessionLen]byte, seq uint32) []byte {
	b := append([]byte{TypeKeepalive}, s[:]...)
	return binary.BigEndian.AppendUint32(b, seq)
}

// Bye frames a session's end.
func Bye(s [SessionLen]byte, reason byte) []byte {
	return append(append([]byte{TypeBye}, s[:]...), reason)
}

// SessionOf reads the session of a DATA/KEEPALIVE/KA_ACK/BYE frame.
func SessionOf(b []byte) (s [SessionLen]byte, ok bool) {
	if len(b) < 1+SessionLen {
		return s, false
	}
	copy(s[:], b[1:])
	return s, true
}

func mac(key []byte, label string, parts ...[]byte) [MACLen]byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(label))
	for _, p := range parts {
		m.Write(p)
	}
	var out [MACLen]byte
	copy(out[:], m.Sum(nil))
	return out
}

// ClientKey is a device's share key (hex), derived by the server from its
// secret, the device's name and its pairing key: a new pairing under the
// same name gets a new one, and a removed device's key opens nothing.
func ClientKey(secret []byte, name, pairingKey string) string {
	m := mac(secret, "tbs1-client", []byte(name), []byte{0}, []byte(pairingKey))
	return hex.EncodeToString(m[:])
}

// PassKey is hub hubID's pass for the sharer whose share key is sharerKey,
// valid until exp (unix seconds). The server hands it to the hub; the
// sharer derives it from its own key and HELLO.
func PassKey(sharerKey []byte, hubID string, exp uint64) []byte {
	m := mac(sharerKey, "tbs1-pass", []byte(hubID), binary.BigEndian.AppendUint64(nil, exp))
	return m[:]
}

// AuthMAC is the hub's proof in AUTH (key: the pass).
func AuthMAC(key []byte, hubNonce, sharerNonce [NonceLen]byte, hubID string) [MACLen]byte {
	return mac(key, "tbs1-auth", hubNonce[:], sharerNonce[:], []byte(hubID))
}

// WelcomeMAC is the sharer's proof in WELCOME.
func WelcomeMAC(key []byte, hubNonce, sharerNonce [NonceLen]byte, s [SessionLen]byte, capLeft uint64) [MACLen]byte {
	return mac(key, "tbs1-welcome", hubNonce[:], sharerNonce[:], s[:], binary.BigEndian.AppendUint64(nil, capLeft))
}

// RefuseMAC authenticates a REFUSE.
func RefuseMAC(key []byte, hubNonce, sharerNonce [NonceLen]byte, reason byte) [MACLen]byte {
	return mac(key, "tbs1-refuse", hubNonce[:], sharerNonce[:], []byte{reason})
}

// KeyID is a short public tag of a sharer's share key (mDNS TXT "k="): the
// hub picks the pass for it, and skips sharers it has none for (another
// home server's, or removed). It reveals nothing of the key.
func KeyID(key []byte) string {
	m := mac(key, "tbs1-keyid")
	return hex.EncodeToString(m[:4])
}

// Equal compares MACs in constant time.
func Equal(a, b [MACLen]byte) bool { return hmac.Equal(a[:], b[:]) }
