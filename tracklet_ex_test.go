package mpcformat_test

import (
	"fmt"

	"github.com/soniakeys/mpcformat"
)

// an observation type for the example, with just enough data to
// satisfy TrackletSplitter.
type mock struct {
	site string
	date string
	mjd  float64
}

func mustMock(date, site string) mock {
	mjd, ok := mpcformat.ParseObs80Date(date)
	if !ok {
		panic(site + " " + date)
	}
	return mock{site, date, mjd}
}

// implement TrackletSplitter
func (m mock) MJD() float64     { return m.mjd }
func (m mock) Observer() string { return m.site }

func ExampleFindTrackletsIndex() {
	// (example data taken from MPEC-B144)
	arc := []mpcformat.TrackletSplitter{
		mustMock("2015 01 23.39252", "F51"),
		mustMock("2015 01 23.40581", "F51"),
		mustMock("2015 01 23.41924", "F51"),
		mustMock("2015 01 23.43258", "F51"),
		mustMock("2015 01 24.218862", "H21"),
		mustMock("2015 01 24.220440", "H01"),
		mustMock("2015 01 24.224126", "H01"),
		mustMock("2015 01 24.224238", "H21"),
		mustMock("2015 01 24.22465", "H36"),
		mustMock("2015 01 24.230395", "H21"),
		mustMock("2015 01 24.234852", "H01"),
		mustMock("2015 01 24.243247", "H01"),
		mustMock("2015 01 24.24584", "H36"),
		mustMock("2015 01 25.16764", "807"),
		mustMock("2015 01 25.168554", "H01"),
		mustMock("2015 01 25.171843", "H01"),
		mustMock("2015 01 25.17513", "807"),
		mustMock("2015 01 25.18295", "807"),
		mustMock("2015 01 25.202440", "H01"),
		mustMock("2015 01 25.212352", "H01"),
		mustMock("2015 01 25.38900", "F51"),
		mustMock("2015 01 25.40205", "F51"),
		mustMock("2015 01 25.41513", "F51"),
		mustMock("2015 01 27.17787", "807"),
		mustMock("2015 01 27.18402", "807"),
	}
	for _, tk := range mpcformat.FindTrackletsIndex(arc) {
		fmt.Println("")
		for _, index := range tk {
			m := arc[index].(mock)
			fmt.Println(m.site, m.date)
		}
	}
	// Output:
	//
	// F51 2015 01 23.39252
	// F51 2015 01 23.40581
	// F51 2015 01 23.41924
	// F51 2015 01 23.43258
	//
	// H21 2015 01 24.218862
	// H21 2015 01 24.224238
	// H21 2015 01 24.230395
	//
	// H01 2015 01 24.220440
	// H01 2015 01 24.224126
	// H01 2015 01 24.234852
	// H01 2015 01 24.243247
	//
	// H36 2015 01 24.22465
	// H36 2015 01 24.24584
	//
	// 807 2015 01 25.16764
	// 807 2015 01 25.17513
	// 807 2015 01 25.18295
	//
	// H01 2015 01 25.168554
	// H01 2015 01 25.171843
	// H01 2015 01 25.202440
	// H01 2015 01 25.212352
	//
	// F51 2015 01 25.38900
	// F51 2015 01 25.40205
	// F51 2015 01 25.41513
	//
	// 807 2015 01 27.17787
	// 807 2015 01 27.18402
}
