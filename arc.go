// Public domain.

package mpcformat

import (
	"bufio"
	"errors"
	"fmt"
	"io"

	"github.com/soniakeys/observation"
)

// ArcSplitter returns a function that splits an observation stream by
// designation, yielding parsed observation arcs.
//
// The stream rObs is a stream of observations in the MPC 80 column format.
// The stream should have observations already grouped by designation.
// That is, this function does not sort or accumulate groups, but simply
// breaks the input stream at designation changes.
//
// Observations are parsed against pMap.
// Read errors should be considered fatal.
// Parse errors are not fatal but do terminate arcs.
func ArcSplitter(rObs io.Reader, pMap observation.ParallaxMap) func() (*observation.Arc, error, error) {
	s := bufio.NewScanner(rObs)
	var desig string       // designation last read
	var o observation.VObs // observation last read
	var a observation.Arc  // arc under construction
	return func() (arc *observation.Arc, readErr, parseErr error) {
		a.Obs = a.Obs[:0]
		arc = &a
		// o != nil means a valid observation was scanned on the previous call.
		if o != nil {
			a.Obs = append(a.Obs, o)
			a.Desig = desig
			o = nil
		}
		for {
			if !s.Scan() {
				if readErr = s.Err(); readErr != nil {
					return
				}
				readErr = io.EOF
			}
			line := s.Text()
			switch {
			case len(line) == 80:
			case len(line) == 0 && readErr == io.EOF:
				o = nil
				return
			default:
				parseErr = fmt.Errorf(
					"observation line length = %d, want 80", len(line))
				return
			}
			if line[14] == 's' {
				s, ok := o.(*observation.SatObs)
				if !ok {
					parseErr = errors.New(
						"space-based observation line 2 without line 1")
					return
				}
				if parseErr = ParseSat2(line, desig, s); parseErr != nil {
					return
				}
			}
			switch desig, o, parseErr = ParseObs80(line, pMap); {
			case parseErr != nil:
				return
			case len(a.Obs) == 0:
				a.Desig = desig // begin new arc
				fallthrough
			case desig == a.Desig:
				a.Obs = append(a.Obs, o) // add observation to arc
			default:
				return // normal return
			}
		}
	}
}
