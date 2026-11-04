package share

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

// testdata/vectors.json pins the wire format and MACs; the Android and iOS
// relays check theirs against the same file (UPDATE_VECTORS=1 rewrites it).
type vectors struct {
	Key         string `json:"key"` // the sharer's share key
	Exp         uint64 `json:"exp"`
	PassKey     string `json:"pass_key"` // the hub's pass: MACs below use it
	HubNonce    string `json:"hub_nonce"`
	SharerNonce string `json:"sharer_nonce"`
	HubID       string `json:"hub_id"`
	SharerID    string `json:"sharer_id"`
	Label       string `json:"label"`
	Session     string `json:"session"`
	CapLeft     uint64 `json:"cap_left"`
	Reason      byte   `json:"reason"`
	KeyID       string `json:"key_id"`
	AuthMAC     string `json:"auth_mac"`
	WelcomeMAC  string `json:"welcome_mac"`
	RefuseMAC   string `json:"refuse_mac"`
	Hello       string `json:"hello"`
	Challenge   string `json:"challenge"`
	Auth        string `json:"auth"`
	Welcome     string `json:"welcome"`
	Refuse      string `json:"refuse"`
	Data        string `json:"data"`
	Keepalive   string `json:"keepalive"`
	KAAck       string `json:"ka_ack"`
	Bye         string `json:"bye"`
}

func buildVectors() vectors {
	key := []byte("tailbond-share-key-for-vectors-0")
	const exp = 1767225600 // 2026-01-01T00:00:00Z
	pass := PassKey(key, "slate7", exp)
	var hn, sn [NonceLen]byte
	for i := range hn {
		hn[i], sn[i] = byte(i), byte(0xa0+i)
	}
	sess := [SessionLen]byte{1, 2, 3, 4, 5, 6, 7, 8}
	h := hex.EncodeToString
	am, wm, rm := AuthMAC(pass, hn, sn, "slate7"), WelcomeMAC(pass, hn, sn, sess, 1<<30), RefuseMAC(pass, hn, sn, ReasonBattery)
	return vectors{
		Key: string(key), Exp: exp, PassKey: h(pass), HubNonce: h(hn[:]), SharerNonce: h(sn[:]), HubID: "slate7", SharerID: "pixel-1", Label: "Pixel's 5G",
		Session: h(sess[:]), CapLeft: 1 << 30, Reason: ReasonBattery, KeyID: KeyID(key),
		AuthMAC: h(am[:]), WelcomeMAC: h(wm[:]), RefuseMAC: h(rm[:]),
		Hello:     h((&Hello{HubNonce: hn, Exp: exp, HubID: "slate7"}).Marshal()),
		Challenge: h((&Challenge{HubNonce: hn, SharerNonce: sn, SharerID: "pixel-1", Label: "Pixel's 5G"}).Marshal()),
		Auth:      h((&Auth{HubNonce: hn, SharerNonce: sn, MAC: am}).Marshal()),
		Welcome:   h((&Welcome{HubNonce: hn, Session: sess, CapLeft: 1 << 30, MAC: wm}).Marshal()),
		Refuse:    h((&Refuse{HubNonce: hn, Reason: ReasonBattery, MAC: rm}).Marshal()),
		Data:      h(Data(nil, sess, []byte("quic"))),
		Keepalive: h(Keepalive(sess, 7)),
		KAAck:     h((&KAAck{Session: sess, Seq: 7, Up: 1000, Down: 2000, CapLeft: NoCap, Battery: 63, Flags: FlagCharging}).Marshal()),
		Bye:       h(Bye(sess, ReasonCap)),
	}
}

func TestVectors(t *testing.T) {
	got := buildVectors()
	b, _ := json.MarshalIndent(got, "", "  ")
	b = append(b, '\n')
	if os.Getenv("UPDATE_VECTORS") == "1" {
		if err := os.WriteFile("testdata/vectors.json", b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile("testdata/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(b) {
		t.Fatalf("wire format changed; testdata/vectors.json (and the Android/iOS relays) must follow:\n%s", b)
	}
}
