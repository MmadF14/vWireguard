package model

import (
	"strings"
	"testing"
)

func TestGenerateAmneziaWGParamsIsValid(t *testing.T) {
	for i := 0; i < 200; i++ {
		p, err := GenerateAmneziaWGParams()
		if err != nil {
			t.Fatal(err)
		}
		if err := p.Validate(); err != nil {
			t.Fatalf("generated params %+v are invalid: %v", p, err)
		}
	}
}

func TestAmneziaWGParamsValidate(t *testing.T) {
	good := AmneziaWGParams{Jc: 4, Jmin: 40, Jmax: 70, S1: 20, S2: 30, H1: 5, H2: 6, H3: 7, H4: 8}
	if err := good.Validate(); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	cases := map[string]func(p *AmneziaWGParams){
		"jc zero":           func(p *AmneziaWGParams) { p.Jc = 0 },
		"jmin >= jmax":      func(p *AmneziaWGParams) { p.Jmin = 70 },
		"jmax too big":      func(p *AmneziaWGParams) { p.Jmax = 1281 },
		"s1 + 56 == s2":     func(p *AmneziaWGParams) { p.S1, p.S2 = 20, 76 },
		"stock header":      func(p *AmneziaWGParams) { p.H1 = 1 },
		"duplicate headers": func(p *AmneziaWGParams) { p.H4 = p.H1 },
		"multiline I1":      func(p *AmneziaWGParams) { p.I1 = "<r 16>\nPostUp = rm -rf /" },
	}
	for name, mutate := range cases {
		p := good
		mutate(&p)
		if err := p.Validate(); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestAmneziaWGConfigLines(t *testing.T) {
	p := AmneziaWGParams{Jc: 4, Jmin: 40, Jmax: 70, S1: 20, S2: 30, H1: 5, H2: 6, H3: 7, H4: 8, I2: "<r 16>"}
	got := p.ConfigLines()
	want := "Jc = 4\nJmin = 40\nJmax = 70\nS1 = 20\nS2 = 30\nH1 = 5\nH2 = 6\nH3 = 7\nH4 = 8\nI2 = <r 16>\n"
	if got != want {
		t.Fatalf("ConfigLines() =\n%s\nwant\n%s", got, want)
	}
	if strings.Contains(got, "I1") {
		t.Fatal("empty I1 must not be written")
	}
}

func TestWGInterfaceAmneziaPaths(t *testing.T) {
	wg := WGInterface{Name: "wg1"}
	if wg.ConfigFilePath() != "/etc/wireguard/wg1.conf" || wg.ServiceName() != "wg-quick@wg1" || wg.Tool() != "wg" {
		t.Fatalf("unexpected WireGuard paths: %s %s %s", wg.ConfigFilePath(), wg.ServiceName(), wg.Tool())
	}
	awg := WGInterface{Name: "awg0", Type: InterfaceTypeAmneziaWG}
	if awg.ConfigFilePath() != "/etc/amnezia/amneziawg/awg0.conf" || awg.ServiceName() != "awg-quick@awg0" || awg.Tool() != "awg" {
		t.Fatalf("unexpected AmneziaWG paths: %s %s %s", awg.ConfigFilePath(), awg.ServiceName(), awg.Tool())
	}
}
