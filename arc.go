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
// Observations are parsed against ocdMap.
// Read errors are relayed on errCh should be considered fatal.
// Parse errors are not fatal.
func ArcSplitter(rObs io.Reader, ocdMap observation.ParallaxMap) func() (arc *observation.Arc, readError, formatError error) {
	s := bufio.NewScanner(rObs)
	var desig string       // designation last read
	var o observation.VObs // observation last read
	var a observation.Arc  // arc under construction
	return func() (arc *observation.Arc, readError, parseError error) {
		a.Obs = a.Obs[:0]
		// o != nil means a valid observation was scanned on the previous call.
		if o != nil {
			a.Obs = append(a.Obs, o)
			a.Desig = desig
		}
		for {
			if !s.Scan() {
				return &a, io.EOF, nil
			}
			err := s.Err()
			if err != nil {
				return &a, err, nil
			}
			line := s.Text()
			if len(line) != 80 {
				return &a, nil, fmt.Errorf(
					"observation line length = %d, want 80", len(line))
			}
			if line[14] == 's' {
				s, ok := o.(*observation.SatObs)
				if !ok {
					return &a, nil, errors.New(
						"space-based observation line 2 without line 1")
				}
				if err := ParseSat2(line, desig, s); err != nil {
					return &a, nil, err
				}
			}
			switch desig, o, err = ParseObs80(line, ocdMap); {
			case err != nil:
				return &a, nil, err // return the parse error
			case len(a.Obs) == 0:
				a.Desig = desig // begin new arc
				fallthrough
			case desig == a.Desig:
				a.Obs = append(a.Obs, o) // add observation to arc
			default:
				return &a, nil, nil // normal return
			}
		}
	}
}
