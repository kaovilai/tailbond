// Package bleproto is the reference for the Bluetooth LE control channel
// between a Mac and a phone running tailbond (docs/phone-hub.md, "One click
// from the Mac"): the Mac asks the phone to start its hotspot and its phone
// hub, then joins the hotspot and uses the hub.
//
// tailbondd doesn't speak Bluetooth itself (the menu bar app does, through
// CoreBluetooth; the phone through its GATT server). This package pins the
// bytes both sides must agree on, and testdata/vectors.json holds them for
// the Kotlin (android/.../ble/BleProtocol.kt) and Swift
// (macos/TailbondMenu/.../PhoneLink.swift) implementations.
//
// The protocol:
//
//   - GATT: service Service; the Mac writes to RX, the phone notifies on TX.
//   - Framing: each message is a 2-byte big-endian length, then that many
//     bytes (at most MaxMessage). The stream is cut into pieces of the ATT
//     payload size (MTU - 3) on both characteristics; the receiver
//     reassembles.
//   - Login: the same pass the phone-hub proxy uses (share.PassKey: the pass
//     the home server gave this Mac for the phone), under another label, so
//     the Bluetooth and the proxy credentials differ:
//     secret = first 16 characters of base32(HMAC-SHA256(pass, "tbp1-ble")),
//     lower case. The user is "<hub>/<expiry>", as for the proxy.
//   - hello (both ways, plain JSON) carries a 16-byte nonce each (cn from the
//     Mac, sn from the phone); the phone's also says its hub id, label and
//     share key id.
//   - auth (Mac, plain): user and AuthMAC; the phone re-derives the pass
//     from its own key and checks it. Its answer (plain) carries PhoneProof,
//     so the Mac knows the phone holds the key too.
//   - After auth every message is sealed: AES-256-GCM with SessionKey, a
//     12-byte nonce of the direction (1 Mac→phone, 2 phone→Mac), three zero
//     bytes and a 64-bit big-endian counter from 0, one per direction. A
//     message that doesn't open ends the session.
//   - Commands (sealed): {"t":"cmd","id":N,"cmd":"..."}; the answer is
//     {"t":"res","id":N,"ok":true,...} or {"ok":false,"code":"...","error":"..."}.
//     hotspot.on, hotspot.off, hotspot.state, hub.on, hub.off, status. The
//     phone may also send {"t":"state",...} when its state changes.
//   - Anything else before auth, or an unknown command, is refused.
package bleproto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/kaovilai/tailbond/agent/internal/share"
)

// GATT UUIDs.
const (
	Service = "5f1d0001-7462-4e64-8f6c-7461696c626f"
	RX      = "5f1d0002-7462-4e64-8f6c-7461696c626f" // Mac → phone: write
	TX      = "5f1d0003-7462-4e64-8f6c-7461696c626f" // phone → Mac: notify
)

// Version is the protocol version in hello.
const Version = 1

// MaxMessage is the longest message, framed or sealed.
const MaxMessage = 4096

// Directions, the first byte of a sealed message's nonce.
const (
	ToPhone byte = 1
	ToMac   byte = 2
)

// Commands.
const (
	CmdHotspotOn    = "hotspot.on"
	CmdHotspotOff   = "hotspot.off"
	CmdHotspotState = "hotspot.state"
	CmdHubOn        = "hub.on"
	CmdHubOff       = "hub.off"
	CmdStatus       = "status"
)

// Commands lists every command a phone serves.
var Commands = []string{CmdHotspotOn, CmdHotspotOff, CmdHotspotState, CmdHubOn, CmdHubOff, CmdStatus}

// Error codes in a refused answer.
const (
	CodeShizuku     = "shizuku"     // Shizuku isn't running, or tailbond may not use it
	CodeUnsupported = "unsupported" // Android older than 11
	CodeHotspot     = "hotspot"     // the system refused to start tethering
	CodeHub         = "hub"         // the phone hub couldn't start (not paired, port taken)
	CodeUnknown     = "unknown"     // not a command
	CodeAuth        = "auth"        // login refused, or a message before login
)

func b32(b []byte) string {
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b))
}

func mac(key []byte, s string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(s))
	return m.Sum(nil)
}

// Credentials is this Mac's Bluetooth login for one phone, from the pass
// the home server gave it for that phone: user "<hub>/<exp>", and the
// secret (16 lower-case base32 characters).
func Credentials(pass []byte, hub string, exp uint64) (user, secret string) {
	return hub + "/" + strconv.FormatUint(exp, 10), b32(mac(pass, "tbp1-ble"))[:16]
}

func transcript(label, cn, sn, user string) string {
	return label + "|" + cn + "|" + sn + "|" + user
}

// AuthMAC is the Mac's proof, hex: HMAC-SHA256(secret, "tbp1-ble-auth|cn|sn|user"),
// cn and sn as sent (base64).
func AuthMAC(secret, cn, sn, user string) string {
	return hex.EncodeToString(mac([]byte(secret), transcript("tbp1-ble-auth", cn, sn, user)))
}

// PhoneProof is the phone's answer, hex: HMAC-SHA256(secret, "tbp1-ble-phone|cn|sn|user").
func PhoneProof(secret, cn, sn, user string) string {
	return hex.EncodeToString(mac([]byte(secret), transcript("tbp1-ble-phone", cn, sn, user)))
}

// SessionKey seals everything after auth: HMAC-SHA256(secret, "tbp1-ble-session|cn|sn|user").
func SessionKey(secret, cn, sn, user string) []byte {
	return mac([]byte(secret), transcript("tbp1-ble-session", cn, sn, user))
}

// Verifier checks a login on the phone, as proxy.Verifier does: the pass
// derived from the phone's own share key for the hub and expiry in user,
// not expired, the hub allowed (allowed nil: any). It returns the secret
// the session goes on with.
func Verifier(ownKey []byte, now func() time.Time, allowed func(hub string) bool) func(user, cn, sn, got string) (secret string, ok bool) {
	return func(user, cn, sn, got string) (string, bool) {
		hub, e, ok := strings.Cut(user, "/")
		if !ok || hub == "" {
			return "", false
		}
		exp, err := strconv.ParseUint(e, 10, 64)
		if err != nil || int64(exp) < now().Unix() {
			return "", false
		}
		if allowed != nil && !allowed(hub) {
			return "", false
		}
		_, secret := Credentials(share.PassKey(ownKey, hub, exp), hub, exp)
		if !hmac.Equal([]byte(got), []byte(AuthMAC(secret, cn, sn, user))) {
			return "", false
		}
		return secret, true
	}
}

func nonce(dir byte, ctr uint64) []byte {
	n := make([]byte, 12)
	n[0] = dir
	binary.BigEndian.PutUint64(n[4:], ctr)
	return n
}

// Seal encrypts one message after auth.
func Seal(key []byte, dir byte, ctr uint64, plain []byte) ([]byte, error) {
	g, err := gcm(key)
	if err != nil {
		return nil, err
	}
	return g.Seal(nil, nonce(dir, ctr), plain, nil), nil
}

// Open decrypts one message after auth.
func Open(key []byte, dir byte, ctr uint64, sealed []byte) ([]byte, error) {
	g, err := gcm(key)
	if err != nil {
		return nil, err
	}
	return g.Open(nil, nonce(dir, ctr), sealed, nil)
}

func gcm(key []byte) (cipher.AEAD, error) {
	b, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(b)
}

// ErrTooLong: a message longer than MaxMessage.
var ErrTooLong = errors.New("bleproto: message too long")

// Frame prefixes msg with its length.
func Frame(msg []byte) ([]byte, error) {
	if len(msg) > MaxMessage {
		return nil, ErrTooLong
	}
	out := make([]byte, 2, 2+len(msg))
	binary.BigEndian.PutUint16(out, uint16(len(msg)))
	return append(out, msg...), nil
}

// Chunks cuts a framed message into ATT payloads for mtu (payload mtu-3,
// at least 20 bytes: the default ATT MTU is 23).
func Chunks(framed []byte, mtu int) [][]byte {
	n := max(mtu-3, 20)
	var out [][]byte
	for len(framed) > 0 {
		k := min(n, len(framed))
		out = append(out, framed[:k])
		framed = framed[k:]
	}
	return out
}

// Reassembler turns received pieces back into messages.
type Reassembler struct {
	buf []byte
}

// Add takes one piece and returns the messages it completes. A length
// over MaxMessage is an error, and the buffer is dropped.
func (r *Reassembler) Add(piece []byte) ([][]byte, error) {
	r.buf = append(r.buf, piece...)
	var out [][]byte
	for len(r.buf) >= 2 {
		n := int(binary.BigEndian.Uint16(r.buf))
		if n > MaxMessage {
			r.buf = nil
			return out, ErrTooLong
		}
		if len(r.buf) < 2+n {
			break
		}
		out = append(out, append([]byte(nil), r.buf[2:2+n]...))
		r.buf = r.buf[2+n:]
	}
	return out, nil
}
