package pairing

import (
	"fmt"
	"strings"

	"rsc.io/qr"
)

// QRSVG renders text as a scalable SVG QR code (quiet zone of 4 modules).
func QRSVG(text string) (string, error) {
	code, err := qr.Encode(text, qr.M)
	if err != nil {
		return "", err
	}
	const quiet = 4
	n := code.Size
	dim := n + 2*quiet
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges">`, dim, dim)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#fff"/><path fill="#000" d="`, dim, dim)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if !code.Black(x, y) {
				continue
			}
			// merge horizontal runs into one rect to keep the SVG small
			run := 1
			for x+run < n && code.Black(x+run, y) {
				run++
			}
			fmt.Fprintf(&b, "M%d %dh%dv1h-%dz", x+quiet, y+quiet, run, run)
			x += run - 1
		}
	}
	b.WriteString(`"/></svg>`)
	return b.String(), nil
}
