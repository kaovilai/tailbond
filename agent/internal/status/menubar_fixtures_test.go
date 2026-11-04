// SPDX-License-Identifier: Apache-2.0

package status

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The Mac menu bar app's test statuses (macos/TailbondMenu) must be what
// this package sends: every field known here, nothing invented.
func TestMenuBarFixturesAreStatuses(t *testing.T) {
	files, _ := filepath.Glob("../../../macos/TailbondMenu/Tests/TailbondMenuKitTests/Fixtures/*.json")
	if len(files) < 4 {
		t.Fatalf("menu bar fixtures: %v", files)
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		d := json.NewDecoder(bytes.NewReader(b))
		d.DisallowUnknownFields()
		var st Status
		if err := d.Decode(&st); err != nil {
			t.Errorf("%s: %v", filepath.Base(f), err)
		}
		if st.Role != "client" || st.Platform != "darwin" {
			t.Errorf("%s: role %q platform %q", filepath.Base(f), st.Role, st.Platform)
		}
	}
}
