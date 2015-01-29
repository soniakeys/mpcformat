// Public domain.

package mpcformat_test

import (
	"bytes"
	"math"
	"testing"

	"github.com/soniakeys/mpcformat"
	"github.com/soniakeys/observation"
)

var ocdSample = `
Code  Long.   cos      sin    Name
000   0.0000 0.62411 +0.77873 Greenwich
248   0.000000.000000 0.000000Hipparcos
250                           Hubble Space Telescope
644 243.140220.836325+0.546877Palomar Mountain/NEAT
703 249.267360.845315+0.533213Catalina Sky Survey
E12 149.0642 0.85563 -0.51621 Siding Spring Survey
`

var pMap, pMapErr = mpcformat.ReadObscodeDat(bytes.NewBufferString(ocdSample))

var siteTestCases = []struct {
	code          string
	lon, cos, sin float64
}{
	{"000", 0, .62411, .77873},
	{"248", 0, 0, 0},
	{"250", 0, 0, 0},
	{"644", 243.14022, .836325, .546877},
	{"E12", 149.0642, .85563, -.51621},
}

func testParallaxMap(m observation.ParallaxMap, t *testing.T) {
	for _, c := range siteTestCases {
		switch s, ok := m[c.code]; {
		case !ok:
			t.Fatal("missing", c.code)
		case s == nil:
			if c.cos != 0 || c.sin != 0 {
				t.Fatal("nil stored for code", c.code)
			}
		case c.cos == 0 && c.sin == 0:
			t.Fatal("expected nil for code", c.code)
		case math.Abs(s.Longitude*360-c.lon) > 1e-10:
			t.Fatal("bad longitude, code", c.code)
		case math.Abs(s.RhoCosPhi*149.59787e9/6.37814e6-c.cos) > 1e-10:
			t.Fatal("bad rho cos, code", c.code)
		case math.Abs(s.RhoSinPhi*149.59787e9/6.37814e6-c.sin) > 1e-10:
			t.Fatal("bad rho sin, code", c.code)
		}
	}
}
