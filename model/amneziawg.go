package model

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

// ---------------------------------------------------------------------------
//  AmneziaWG support
//
//  Plain WireGuard is trivial for DPI to fingerprint: the handshake packets
//  have fixed sizes (148 / 92 bytes) and fixed message-type headers (1..4).
//  AmneziaWG (https://github.com/amnezia-vpn/amneziawg-go) is a WireGuard fork
//  that breaks those signatures:
//
//    Jc, Jmin, Jmax  junk packets of random size sent before the handshake
//    S1, S2          random padding prepended to handshake init / response
//    H1..H4          custom values replacing the four message-type headers
//    I1..I5          optional "signature" packets that imitate another
//                    protocol (QUIC, DNS, ...) - AmneziaWG 1.5+ only
//
//  The server and every client on an interface MUST share S1, S2 and H1..H4,
//  otherwise the handshake never completes. Jc/Jmin/Jmax/I1..I5 only affect
//  the sending side but are written to both so the configs stay symmetrical.
//
//  An AmneziaWG interface is driven by awg / awg-quick instead of wg /
//  wg-quick, keeps its config under /etc/amnezia/amneziawg/ and is invisible
//  to wgctrl, so callers branch on WGInterface.IsAmnezia().
// ---------------------------------------------------------------------------

const (
	// InterfaceTypeWireGuard is plain WireGuard. Empty means the same thing,
	// which is what every record written before AmneziaWG support has.
	InterfaceTypeWireGuard = "wireguard"
	// InterfaceTypeAmneziaWG is an AmneziaWG interface.
	InterfaceTypeAmneziaWG = "amneziawg"
)

// Limits from the AmneziaWG documentation. 1280 is the IPv6 minimum MTU; the
// padded handshake (148+S1 / 92+S2) and the junk packets must fit inside it.
const (
	awgMaxJc         = 128
	awgMaxJunkSize   = 1280
	awgMaxS1         = 1280 - 148
	awgMaxS2         = 1280 - 92
	awgMinHeader     = 5 // 1..4 are the stock WireGuard message types
	awgMaxHeader     = 2147483647
	awgInitRespDelta = 56 // 148 - 92: S1+56 == S2 makes both packets the same size again
)

// AmneziaWGParams are the obfuscation settings of an AmneziaWG interface.
type AmneziaWGParams struct {
	Jc   int `json:"jc"`
	Jmin int `json:"jmin"`
	Jmax int `json:"jmax"`
	S1   int `json:"s1"`
	S2   int `json:"s2"`
	// H1..H4 are uint32 on the wire; int64 keeps the JSON readable everywhere.
	H1 int64 `json:"h1"`
	H2 int64 `json:"h2"`
	H3 int64 `json:"h3"`
	H4 int64 `json:"h4"`

	// I1..I5 are optional signature packets (AmneziaWG 1.5+), e.g.
	// "<b 0x...><r 16><t>". Empty ones are simply not written, so configs stay
	// compatible with AmneziaWG 1.0 clients.
	I1 string `json:"i1,omitempty"`
	I2 string `json:"i2,omitempty"`
	I3 string `json:"i3,omitempty"`
	I4 string `json:"i4,omitempty"`
	I5 string `json:"i5,omitempty"`
}

// Validate checks the parameters against the AmneziaWG constraints.
func (p *AmneziaWGParams) Validate() error {
	if p.Jc < 1 || p.Jc > awgMaxJc {
		return fmt.Errorf("Jc must be between 1 and %d", awgMaxJc)
	}
	if p.Jmin < 0 || p.Jmax > awgMaxJunkSize || p.Jmin >= p.Jmax {
		return fmt.Errorf("junk sizes must satisfy 0 <= Jmin < Jmax <= %d", awgMaxJunkSize)
	}
	if p.S1 < 0 || p.S1 > awgMaxS1 {
		return fmt.Errorf("S1 must be between 0 and %d", awgMaxS1)
	}
	if p.S2 < 0 || p.S2 > awgMaxS2 {
		return fmt.Errorf("S2 must be between 0 and %d", awgMaxS2)
	}
	if p.S1+awgInitRespDelta == p.S2 {
		return fmt.Errorf("S1 + %d must not equal S2: both handshake packets would have the same size", awgInitRespDelta)
	}
	seen := map[int64]string{}
	for i, h := range []int64{p.H1, p.H2, p.H3, p.H4} {
		name := fmt.Sprintf("H%d", i+1)
		if h < awgMinHeader || h > awgMaxHeader {
			return fmt.Errorf("%s must be between %d and %d", name, awgMinHeader, awgMaxHeader)
		}
		if other, dup := seen[h]; dup {
			return fmt.Errorf("%s and %s must be different", other, name)
		}
		seen[h] = name
	}
	for i, s := range []string{p.I1, p.I2, p.I3, p.I4, p.I5} {
		if strings.ContainsAny(s, "\r\n") {
			return fmt.Errorf("I%d must be a single line", i+1)
		}
	}
	return nil
}

// ConfigLines renders the parameters as [Interface] lines, in the order
// awg-quick prints them.
func (p *AmneziaWGParams) ConfigLines() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Jc = %d\nJmin = %d\nJmax = %d\n", p.Jc, p.Jmin, p.Jmax)
	fmt.Fprintf(&b, "S1 = %d\nS2 = %d\n", p.S1, p.S2)
	fmt.Fprintf(&b, "H1 = %d\nH2 = %d\nH3 = %d\nH4 = %d\n", p.H1, p.H2, p.H3, p.H4)
	for i, s := range []string{p.I1, p.I2, p.I3, p.I4, p.I5} {
		if s = strings.TrimSpace(s); s != "" {
			fmt.Fprintf(&b, "I%d = %s\n", i+1, s)
		}
	}
	return b.String()
}

// GenerateAmneziaWGParams returns a fresh random parameter set inside the
// ranges the AmneziaWG authors recommend (Jc 4-12, Jmin/Jmax 8-1280, S1/S2
// 15-150), so each server gets its own fingerprint.
func GenerateAmneziaWGParams() (AmneziaWGParams, error) {
	var p AmneziaWGParams
	var err error
	pick := func(min, max int64) int64 {
		if err != nil {
			return min
		}
		var n *big.Int
		n, err = rand.Int(rand.Reader, big.NewInt(max-min+1))
		if err != nil {
			return min
		}
		return min + n.Int64()
	}

	p.Jc = int(pick(4, 12))
	p.Jmin = int(pick(8, 64))
	p.Jmax = int(pick(int64(p.Jmin)+64, awgMaxJunkSize))
	p.S1 = int(pick(15, 150))
	for {
		p.S2 = int(pick(15, 150))
		if p.S1+awgInitRespDelta != p.S2 || err != nil {
			break
		}
	}

	headers := map[int64]bool{}
	for len(headers) < 4 && err == nil {
		headers[pick(awgMinHeader, awgMaxHeader)] = true
	}
	if err != nil {
		return AmneziaWGParams{}, fmt.Errorf("cannot generate AmneziaWG parameters: %v", err)
	}
	h := make([]int64, 0, 4)
	for v := range headers {
		h = append(h, v)
	}
	p.H1, p.H2, p.H3, p.H4 = h[0], h[1], h[2], h[3]
	return p, nil
}
