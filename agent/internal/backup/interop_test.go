package backup

import (
	"bytes"
	"os"
	"testing"
)

// The Android app's interop vectors (android/app/src/test/resources/interop):
// the Go side must read the app's backup.
const interop = "../../../android/app/src/test/resources/interop/"

func TestAndroidInterop(t *testing.T) {
	data, err := os.ReadFile(interop + "android.tbbackup")
	if err != nil {
		t.Skip("no Android interop vectors in this checkout")
	}
	h, pt, err := Open(data, "tailbond interop passphrase")
	if err != nil {
		t.Fatalf("open the app's backup: %v", err)
	}
	want, _ := os.ReadFile(interop + "android-backup-plaintext.json")
	if h.Platform != "android" || !bytes.Equal(bytes.TrimSpace(pt), bytes.TrimSpace(want)) {
		t.Fatalf("header %+v, plaintext differs", h)
	}
}
