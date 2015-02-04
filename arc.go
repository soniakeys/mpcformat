// Public domain.

package mpcformat

import (
	"bufio"
	"errors"
	"fmt"
	"io"

	"github.com/soniakeys/observation"
)

type ArcError struct{ error }

// ArcSplitter returns a function that splits an observation stream by
// designation, yielding parsed observation arcs.
//
// The stream rObs is a stream of observations in the MPC 80 column format.
// The stream should have observations already grouped by designation.
// That is, this function does not sort or accumulate groups, but simply
// breaks the input stream at designation changes.
//
// Observations are parsed against pMap.
//
// - If a valid arc is returned, err will be nil.
//
// - After all arcs are returned, err will be io.EOF.
//
// - An err that can be type asserted to type ArcError represents a parse error
// but is not fatal.  The split function can be called again.
//
// - Other errors should be considered fatal and the split function should not
// be called again.
func ArcSplitter(rObs io.Reader, pMap observation.ParallaxMap) func() (*observation.Arc, error) {
	s := bufio.NewScanner(rObs)
	var a observation.Arc // arc under construction
	var (                 // values that may be carried from last call
		desig string
		o     observation.VObs
		err   error
	)
	return func() (*observation.Arc, error) {
		if err != nil { // error from last call
			e := err
			err = nil
			return nil, e
		}
		a.Obs = a.Obs[:0]
		if o != nil { // observation from last call
			a.Desig = desig
			a.Obs = append(a.Obs, o)
		}
	arc:
		for {
			scanOk := s.Scan()
			if !scanOk {
				if err = s.Err(); err != nil {
					return nil, err
				}
			}
			line := s.Text()
			switch len(line) {
			case 80:
			case 0:
				if !scanOk {
					if len(a.Obs) == 0 {
						return nil, io.EOF
					}
					err = io.EOF
					o = nil
					return &a, nil
				}
				fallthrough
			default:
				err = ArcError{fmt.Errorf("observation line length = %d, want 80",
					len(line))}
				break arc
			}
			if line[14] == 's' {
				s, ok := o.(*observation.SatObs)
				if !ok {
					err = ArcError{errors.New(
						"space-based observation line 2 without line 1")}
					break arc
				}
				if err = ParseSat2(line, desig, s); err != nil {
					// TODO maybe back off that last S obs too?
					break arc
				}
				continue // (it's already in the list)
			}
			switch desig, o, err = ParseObs80(line, pMap); {
			case err != nil:
				err = ArcError{err}
				break arc
			case len(a.Obs) == 0:
				a.Desig = desig // begin new arc
				fallthrough
			case desig == a.Desig:
				a.Obs = append(a.Obs, o) // add observation to arc
			default:
				return &a, nil // carry desig, o to next call
			}
		}
		// there was a parse error
		o = nil // (anything there is no good)
		if len(a.Obs) > 0 {
			return &a, nil // return good obs, carry err to next call
		}
		e := err // return err now
		err = nil
		return &a, e
	}
}
