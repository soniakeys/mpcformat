// Public domain.

package mpcformat_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/soniakeys/coord"
	"github.com/soniakeys/mpcformat"
	"github.com/soniakeys/observation"
)

func ExampleParseObs80Date() {
	fmt.Println(mpcformat.ParseObs80Date("2014 09 04.8"))
	// Output:
	// 56904.8 true
}

func TestSiteObs(t *testing.T) {
	if pMapErr != nil {
		t.Skip(pMapErr)
	}
	const obs = "     K11Q14F  C2014 09 03.40285 02 53 00.70 +10 38 30.3          19.2 VqER031703"
	desig, o, err := mpcformat.ParseObs80(obs, pMap)
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
			MJD: 56903.40285,
			Equa: coord.Equa{
				RA:  0.7549058069240641,
				Dec: 0.1857335756741066},
			VMag: 19.2,
			Qual: "703"},
		Par: pMap["703"],
	}
	if math.Abs(so.VMeas.MJD-want.VMeas.MJD) > 1e-6 ||
		math.Abs((so.VMeas.Equa.RA-want.VMeas.Equa.RA).Rad()) > 1e-8 ||
		math.Abs((so.VMeas.Equa.Dec-want.VMeas.Equa.Dec).Rad()) > 1e-8 ||
		math.Abs(so.VMeas.VMag-want.VMeas.VMag) > 1e-2 ||
		so.VMeas.Qual != want.VMeas.Qual ||
		so.Par != want.Par {
		t.Fatalf("ParseObs80 obs = %+v, want %+v", so, want)
	}
}

const (
	tcSatLine1 = "03620         S1996 08 30.51477 21 07 31.918-05 22 00.82                27764250"
	tcSatLine2 = "03620         s1996 08 30.51477 1 -  344.3553 - 6919.1239 +  872.2948   27764250"
)

func TestSatObs(t *testing.T) {
	if pMapErr != nil {
		t.Skip(pMapErr)
	}
	desig, o, err := mpcformat.ParseObs80(tcSatLine1, pMap)
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
	if err = mpcformat.ParseSat2(tcSatLine2, desig, so); err != nil {
		t.Fatal(err)
	}
	want := &observation.SatObs{
		Sat: "250",
		VMeas: observation.VMeas{
			MJD:  50325.51477,
			Equa: coord.Equa{RA: 5.530651548153087, Dec: -0.09366997866254745},
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
		math.Abs((so.VMeas.Equa.RA-want.VMeas.Equa.RA).Rad()) > 1e-8 ||
		math.Abs((so.VMeas.Equa.Dec-want.VMeas.Equa.Dec).Rad()) > 1e-8 ||
		math.Abs(so.VMeas.VMag-want.VMeas.VMag) > 1e-2 ||
		so.VMeas.Qual != want.VMeas.Qual ||
		math.Abs(so.Offset.X-want.Offset.X) > 1e-8 ||
		math.Abs(so.Offset.Y-want.Offset.Y) > 1e-8 ||
		math.Abs(so.Offset.Z-want.Offset.Z) > 1e-8 {
		t.Fatalf("ParseSat2 obs = %+v, want %+v", so, want)
	}
}
