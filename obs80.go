// Public domain.

package mpcformat

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/soniakeys/coord"
	"github.com/soniakeys/observation"
)

// ParseObs80 parses a single line observation in the MPC 80 column format.
//
// Input line80 must be a string of 80 characters.  Other lengths are an error.
// The observatory code in columns 78-80 must exist in the input map.
func ParseObs80(line80 string, ocm observation.ParallaxMap) (desig string,
	o observation.VObs, err error) {
	if len(line80) != 80 {
		err = errors.New("ParseObs80 requires 80 characters")
		return
	}

	// the intent of reallocating desig (and later, obscode) is
	// to allow line80 to be garbage collected sooner. no idea it really helps.
	desig = string([]byte(strings.TrimSpace(line80[:12])))

	d := line80[15:32]
	mjd, ok := ParseObs80Date(d)
	if !ok {
		err = fmt.Errorf("ParseObs80: Invalid date (%s)", d)
		return
	}

	var rah, ram int
	var ras float64
	rah, err = strconv.Atoi(strings.TrimSpace(line80[32:34]))
	if err == nil {
		ram, err = strconv.Atoi(strings.TrimSpace(line80[35:37]))
		if err == nil {
			ras, err =
				strconv.ParseFloat(strings.TrimSpace(line80[38:44]), 64)
		}
	}
	if err != nil {
		err = fmt.Errorf("ParseObs80: Invalid RA (%s), %v", line80[32:44], err)
		return
	}

	decg := line80[44] // minus sign
	var decd, decm int
	var decs float64
	decd, err = strconv.Atoi(strings.TrimSpace(line80[45:47]))
	if err == nil {
		decm, err = strconv.Atoi(strings.TrimSpace(line80[48:50]))
		if err == nil {
			decs, err =
				strconv.ParseFloat(strings.TrimSpace(line80[51:56]), 64)
		}
	}
	if err != nil {
		err = fmt.Errorf("ParseObs80: Invalid Dec (%s), %v", line80[44:56], err)
		return
	}

	var mag float64
	if ts := strings.TrimSpace(line80[65:70]); len(ts) != 0 {
		mag, err = strconv.ParseFloat(ts, 64)
		if err != nil {
			err = fmt.Errorf("ParseObs80: Invalid mag (%s), %v", ts, err)
			return
		}
		band := line80[70]
		switch band {
		case 'V':
			break
		case 'B':
			mag -= .8
		default:
			mag += .4
		}
	}

	c := line80[77:80]
	par, ok := ocm[c]
	if !ok {
		return "", nil,
			fmt.Errorf("ParseObs80: Unknown observatory code (%s)", c)
	}

	obscode := string([]byte(line80[77:80]))

	if par == nil || line80[14] == 'S' {
		o = &observation.SatObs{Sat: obscode}
	} else {
		o = &observation.SiteObs{Par: par}
	}
	m := o.Meas()
	m.MJD = mjd
	m.RA = (float64(rah*60+ram)*60 + ras) * math.Pi / (12 * 3600)
	m.Dec = (float64(decd*60+decm)*60 + decs) * math.Pi / (180 * 3600)
	if decg == '-' {
		m.Dec = -m.Dec
	}
	m.VMag = mag
	// could be enhanced to store program code, eg.  if so, see obsErr
	// code in digest2.readConfig and make appropriate changes.
	m.Qual = obscode
	return
}

var flookup = [13]int{0, 306, 337, 0, 31, 61, 92, 122, 153, 184, 214, 245, 275}

// ParseObs80Date parses a date in the format used in 80 column observation
// records.
//
// The argument should be a string of at least 10 characters with the format
// "yyyy mm dd".  Longer strings allow for a decimal date. (The full width
// of the date field in the observation record is 17.)
//
// Modified Julian date is returned.
func ParseObs80Date(d string) (mjd float64, ok bool) {
	if len(d) < 10 {
		return 0, false
	}
	year, err := strconv.Atoi(d[:4])
	if err != nil {
		return 0, false
	}
	df := d[5:7]
	// allow single digit day.
	// there's little harm in allowing this non-standard variation.
	if df[0] == ' ' {
		df = df[1:]
	}
	month, err := strconv.Atoi(d[5:7])
	if err != nil {
		return 0, false
	}
	day, err := strconv.ParseFloat(strings.TrimSpace(d[8:]), 64)
	if err != nil {
		return 0, false
	}
	z := year + (month-14)/12
	m := flookup[month] + 365*z + z/4 - z/100 + z/400 - 678882
	return float64(m) + day, true
}

// ParseSat2 parses the second line of a space-based observation.
//
// Arguments des1 and s1 must be results of parsing the first line.
// ParseSat2 validates that identifying data matches line 1 and then
// updates s1 with line 2 information.
func ParseSat2(line80, des1 string, s1 *observation.SatObs) error {
	if desig := strings.TrimSpace(line80[:12]); desig != des1 {
		return fmt.Errorf("sat obs line 2 designation = %s, line 1 was %s",
			desig, des1)
	}
	d := line80[15:32]
	switch date2, ok := ParseObs80Date(d); {
	case !ok:
		return fmt.Errorf("sat obs line 2 invalid date (%s)", d)
	case date2 != s1.MJD:
		return fmt.Errorf("sat obs line 2 date %s different from line 1", d)
	}
	if line80[77:80] != s1.Sat {
		return fmt.Errorf("sat obs line 2 obscode = %s, line 1 was %s",
			line80[77:80], s1.Sat)
	}

	x, ok := parseMpcOffset(line80[34:46])
	if !ok {
		return fmt.Errorf("sat obs line 2 invalid offset: %s", line80[34:46])
	}
	y, ok := parseMpcOffset(line80[46:58])
	if !ok {
		return fmt.Errorf("sat obs line 2 invalid offset: %s", line80[46:58])
	}
	z, ok := parseMpcOffset(line80[58:70])
	if !ok {
		return fmt.Errorf("sat obs line 2 invalid offset: %s", line80[58:70])
	}
	if line80[32] == '1' {
		// Scale factor = 1 / 1 AU in km.
		const sf = 1 / 149.59787e6
		x *= sf
		y *= sf
		z *= sf
	}
	s1.Offset = coord.Cart{X: x, Y: y, Z: z}
	return nil
}

func parseMpcOffset(off string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(off[1:]), 64)
	switch {
	case err != nil:
		break
	case off[0] == '-':
		return -v, true
	case off[0] == '+' || off[0] == ' ':
		return v, true
	}
	return 0, false
}
