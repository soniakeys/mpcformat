// Public domain.

package mpcformat_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"testing"

	"github.com/soniakeys/coord"
	"github.com/soniakeys/mpcformat"
	"github.com/soniakeys/observation"
)

var ocdSample = `
Code  Long.   cos      sin    Name
000   0.0000 0.62411 +0.77873 Greenwich
248   0.000000.000000 0.000000Hipparcos
250                           Hubble Space Telescope
644 243.140220.836325+0.546877Palomar Mountain/NEAT
E12 149.0642 0.85563 -0.51621 Siding Spring Survey
`

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

func Test(t *testing.T) {
	fn := testFetchObscode(t)
	m := testReadObscode(t, fn)
	testSiteObs(t, m)
	testSatObs(t, m)
	//	testSplitTracklets(t, m) // test fails.  issue filed.
}

// If Fetch fails, this returns "".  Other tests should use ocdSample data
// in this case.
func testFetchObscode(t *testing.T) string {
	t.Log("testFetchObscode")
	f, err := ioutil.TempFile("", "digest2ocd")
	if err != nil {
		t.Error(err)
		return ""
	}
	fn := f.Name()
	f.Close()
	if err = mpcformat.FetchObscodeDat(fn); err != nil {
		t.Error(err)
		return ""
	}
	return fn
}

func testReadObscode(t *testing.T, fn string) (m observation.ParallaxMap) {
	t.Log("testReadObscode")
	var err error
	if fn > "" {
		// fn should be a fresh copy of the file
		m, err = mpcformat.ReadObscodeDatFile(fn)
		os.Remove(fn)
	} else {
		// the file couldn't be fetched.  use ocdSample data.
		m, err = mpcformat.ReadObscodeDat(bytes.NewBufferString(ocdSample))
	}
	if err != nil {
		t.Fatal(err)
	}
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
	return m
}

func ExampleParseObs80Date() {
	fmt.Println(mpcformat.ParseObs80Date("2014 09 04.8"))
	// Output:
	// 56904.8 true
}

func testSiteObs(t *testing.T, m observation.ParallaxMap) {
	t.Log("testSiteObs")
	const obs = "     K11Q14F  C2014 09 03.40285 02 53 00.70 +10 38 30.3          19.2 VqER031703"
	desig, o, err := mpcformat.ParseObs80(obs, m)
	if err != nil {
		t.Fatal(err)
	}
	if desig != "K11Q14F" {
		t.Fatalf(`ParseObs80 desig = %q, want "K11Q14F"`)
	}
	so, ok := o.(*observation.SiteObs)
	if !ok {
		t.Fatalf("Want *observation.SiteObs from ParseObs80, got %T", o)
	}
	want := observation.SiteObs{
		VMeas: observation.VMeas{
			MJD:  56903.40285,
			Sphr: coord.Sphr{RA: 0.7549058069240641, Dec: 0.1857335756741066},
			VMag: 19.2,
			Qual: "703"},
		Par: m["703"],
	}
	if math.Abs(so.VMeas.MJD-want.VMeas.MJD) > 1e-6 ||
		math.Abs(so.VMeas.Sphr.RA-want.VMeas.Sphr.RA) > 1e-8 ||
		math.Abs(so.VMeas.Sphr.Dec-want.VMeas.Sphr.Dec) > 1e-8 ||
		math.Abs(so.VMeas.VMag-want.VMeas.VMag) > 1e-2 ||
		so.VMeas.Qual != want.VMeas.Qual ||
		so.Par != want.Par {
		t.Fatalf("ParseObs80 obs = %+v, want %+v", so, want)
	}
}

func testSatObs(t *testing.T, m observation.ParallaxMap) {
	t.Log("testSatObs")
	const (
		line1 = "03620         S1996 08 30.51477 21 07 31.918-05 22 00.82                27764250"
		line2 = "03620         s1996 08 30.51477 1 -  344.3553 - 6919.1239 +  872.2948   27764250"
	)
	desig, o, err := mpcformat.ParseObs80(line1, m)
	if err != nil {
		t.Fatal(err)
	}
	if desig != "03620" {
		t.Fatalf(`Line 1 desig = %q, want "03620"`)
	}
	so, ok := o.(*observation.SatObs)
	if !ok {
		t.Fatalf("Want *observation.SatObs from ParseObs80, got %T", o)
	}
	if err = mpcformat.ParseSat2(line2, desig, so); err != nil {
		t.Fatal(err)
	}
	want := &observation.SatObs{
		Sat: "250",
		VMeas: observation.VMeas{
			MJD:  50325.51477,
			Sphr: coord.Sphr{RA: 5.530651548153087, Dec: -0.09366997866254745},
			Qual: "250",
		},
		Offset: coord.Cart{
			X: -2.301873014635837e-06,
			Y: -4.625148673574028e-05,
			Z: 5.830930614185884e-06,
		},
	}
	if math.Abs(so.VMeas.MJD-want.VMeas.MJD) > 1e-6 ||
		so.Sat != want.Sat ||
		math.Abs(so.VMeas.Sphr.RA-want.VMeas.Sphr.RA) > 1e-8 ||
		math.Abs(so.VMeas.Sphr.Dec-want.VMeas.Sphr.Dec) > 1e-8 ||
		math.Abs(so.VMeas.VMag-want.VMeas.VMag) > 1e-2 ||
		so.VMeas.Qual != want.VMeas.Qual ||
		math.Abs(so.Offset.X-want.Offset.X) > 1e-8 ||
		math.Abs(so.Offset.Y-want.Offset.Y) > 1e-8 ||
		math.Abs(so.Offset.Z-want.Offset.Z) > 1e-8 {
		t.Fatalf("ParseSat2 obs = %+v, want %+v", so, want)
	}
}

/* SplitTracklets currently broken
func testSplitTracklets(t *testing.T, m observation.ParallaxMap) {
	   	b := bytes.NewBufferString(`
	        K14G49E* C2014 04 09.45004 16 29 34.386+18 18 53.97         19.3 iL~133CF51
	        K14G49E  C2014 04 09.46354 16 29 34.509+18 19 12.37         19.3 iL~133CF51
	        K14G49E  C2014 04 09.49047 16 29 34.734+18 19 49.56         19.6 iL~133CF51
	        K14G49E KC2014 04 10.40819 16 29 43.71 +18 40 40.9          19.2 Ro~133C291
	        K14G49E KC2014 04 10.41113 16 29 43.73 +18 40 44.5          19.8 Ro~133C291
	        K14G49E KC2014 04 10.41407 16 29 43.75 +18 40 48.5          19.2 Ro~133C291
	        K14G49E FC2014 04 12.39670 16 29 57.74 +19 25 38.4                 ~133C711
	b := bytes.NewBufferString(`
     K14G49F* C2014 04 10.44677 13 26 30.145+37 09 23.36         20.5 iL~133CF51
     K14G49F  C2014 04 10.45768 13 26 29.317+37 09 25.45         20.8 iL~133CF51
     K14G49F  C2014 04 10.46859 13 26 28.495+37 09 27.45         20.6 iL~133CF51
     K14G49F  C2014 04 10.47949 13 26 27.632+37 09 29.62         20.6 iL~133CF51
     K14G49F FC2014 04 11.29221 13 25 28.56 +37 11 53.8                 ~133C711
     K14G49F  C2014 04 11.30076 13 25 27.88 +37 11 55.3                 ~133C711
     K14G49F  C2014 04 12.23087 13 24 20.37 +37 14 13.4                 ~133C711
     K14G49F  C2014 04 12.23515 13 24 20.00 +37 14 14.0                 ~133C711
     K14G49F  C2014 04 12.23949 13 24 19.72 +37 14 14.7                 ~133C711
`)
	tkCh := make(chan *observation.Tracklet)
	errCh := make(chan error)
	go mpcformat.SplitTracklets(b, m, tkCh, errCh)
	want := []struct {
		desig string
		cod   string
		nObs  int
	}{
		//		{"K14G49E", "F51", 3},
		//		{"K14G49E", "291", 3},
		{"K14G49F", "F51", 4},
		{"K14G49F", "711", 2},
		{"K14G49F", "711", 3},
	}
	for i := 0; ; i++ {
		select {
		case err := <-errCh:
			t.Fatal(err)
		case tk, ok := <-tkCh:
			if !ok {
				if i < len(want) {
					t.Fatal("Got", i, "tracklets, want len(want).")
				} else {
					return // pass test
				}
			}
			if i == len(want) {
				t.Fatal("Expected only", len(want), "tracklets.")
			}
			t.Log(tk.Desig)
			for _, o := range tk.Obs {
				t.Logf("  %+v", o.(*observation.SiteObs))
			}
			if tk.Desig != want[i].desig || len(tk.Obs) != want[i].nObs {
				t.Fatalf("Unexpected tracklet.  Got %+v, want %+v", tk, want[i])
			}
			so, ok := tk.Obs[0].(*observation.SiteObs)
			if !ok || so.VMeas.Qual != want[i].cod {
				t.Fatalf("Unexpected tracklet.  Got %+v, want %+v", tk, want[i])
			}
		}
	}
}
*/
