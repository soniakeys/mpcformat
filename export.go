// Public domain.

package mpcformat

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// Unmarshaller for MPC "export format", the format of MPCORB.DAT.
//
// This format will be called "text format" in comments in this file.
// To disambiguate "field," a field of the text format will be called a
// tField; a field of a corresponding Go struct will be called an sField.

// Tags:  recognized struct field tag keys are `val` and `export`.
//
// The val tag specifies things about the struct field and so is not
// file format specific.  Tag values implemented:
//
// defNaN - on any float field, indicates that a blank field in the text
//          format is not an error but instead defaults to NaN.
// deg, rad - on MA, Peri, Node, Inc, M, means to return the result in
//            degrees or radians.  Note that the native format is degrees.
//            Specifying deg is a no-op, and degrees is the default if
//            no unit is specified. (For M this is angle unit per day.)
// Unrecognized values of the `val` key are ignored.
//
// The export key is used to specify an export field name, or to specify
// to ignore the struct field.  Valid forms are:
//    Field    where Field is a map key from tFieldMap, below
//    -        to ignore the struct field
//    -,Field  to "comment out" a field
// Unrecognized values of the `export` key are an error.

/* additional `val` units to implement, maybe...
const (
	Packed = iota
	NotPacked
	ArcSec
	AU
	Days
	J2000
	MJD
)
*/

// Decode data for fields of text format.
// Start and end are column numbers, Go-like numbering.
// Terp is one of the constants below
type decodeData struct {
	start, end, terp int
}

// Terp specifies how to interpret a tField before storing it in an sField.
// Two kinds of limitations are checked:
// 1.  a tField must be interpreted in a meaningful way.
// 2.  the interpretation must be compatible with the sField type.
const (
	terpString = iota
	terpFloat
	terpInt
	terpBool
	terpByte
	// sField type can be time.Time.  J2000 values will be stored in
	// int or float sFields.  string sFields get the (trimmed) tField.
	terpDate
)

// Fields of the text representation.  Decode data is mapped to a field name.
// Terp values here represent the strictest way to interpret a field.
var tFieldMap = map[string]decodeData{
	"Desig":   {0, 7, terpString},     // Number or provisional designation
	"Num":     {0, 7, terpInt},        // Numbered object designation
	"Prov":    {0, 7, terpString},     // Provisional designation
	"H":       {8, 13, terpFloat},     // Absolute magnitude, H
	"G":       {14, 19, terpFloat},    // Slope parameter, G
	"Epoch":   {20, 25, terpDate},     // Epoch
	"MA":      {26, 35, terpFloat},    // Mean anomaly at the epoch
	"Peri":    {37, 46, terpFloat},    // Argument of perihelion
	"Node":    {48, 57, terpFloat},    // Longitude of the ascending node
	"Inc":     {59, 68, terpFloat},    // Inclination to the ecliptic
	"E":       {70, 79, terpFloat},    // Orbital eccentricity
	"M":       {80, 91, terpFloat},    // Mean daily motion
	"A":       {92, 103, terpFloat},   // Semimajor axis
	"U":       {105, 106, terpInt},    // Uncertainty parameter
	"EAsm":    {105, 106, terpBool},   // E-assumed
	"DD":      {105, 106, terpBool},   // double or multiple designation
	"Ref":     {107, 116, terpString}, // Reference
	"NObs":    {117, 122, terpInt},    // Number of observations
	"NOpp":    {123, 126, terpInt},    // Number of oppositions
	"YFirst":  {127, 131, terpInt},    // Year of first observation
	"YLast":   {132, 136, terpInt},    // Year of last observation
	"Arc":     {127, 131, terpInt},    // Arc length
	"RMS":     {137, 141, terpFloat},  // r.m.s residual
	"Coarse":  {142, 145, terpString}, // perturbers by coarse indicator
	"Precise": {146, 148, terpInt},    // perturbers by precise indicator
	"Ptb":     {142, 149, terpInt},    // combined, per bits defined below
	// PlEph as a byte is the raw "system descriptor" per "Perturbers.html"
	// as a string it is expanded into the printable "JPL DExxx" format.
	"PlEph":       {148, 149, terpByte},
	"Comp":        {150, 160, terpString}, // agent which computed orbit
	"Type":        {163, 165, terpInt},    // per orbit type constants below
	"NEO":         {162, 163, terpBool},   // object is NEO
	"Km":          {161, 162, terpBool},   // object is 1-km (or larger) NEO
	"Seen":        {161, 162, terpBool},   // "...seen at earlier opposition"
	"Crit":        {161, 162, terpBool},   // Critical list numbered object
	"PHA":         {161, 162, terpBool},   // true means PHA
	"Designation": {166, 194, terpString}, // Readable designation
	// date of last observation used in orbit solution
	"LastObs": {194, 202, terpDate},
}

// Ptb bits consist of "precise" and "planetary" bits.

// Export format precise perturber bit definitions, per "precise indicator"
// bit defintions in MPC document "Perturbers.html."
const (
	ExHygiea = 1 << iota
	ExEarth
	ExMoon
	ExCeres
	ExPallas
	ExVesta
	ExEunomia
)

// Export format planetary perturber bit definitions.
// These encode additional perturbers implied with use of precise indicators,
// and specified with use of "coarse indicators."
const (
	ExMercury = 1 << (iota + 16)
	ExVenus
	ExEMBary
	ExMars
	ExJupiter
	ExSaturn
	ExUranus
	ExNeptune
	ExPluto
)

// Export format orbit types for 'Type' field
const (
	ExAten     = 2
	ExApollo   = 3
	ExAmor     = 4
	ExMC       = 5 // Object with q < 1.665 AU
	ExHungaria = 6
	ExPhocaea  = 7
	ExHilda    = 8
	ExTrojan   = 9 // Jupiter Trojan
	ExCentaur  = 10
	ExPlutino  = 14
	ExTNO      = 15 // Other resonant TNO
	ExCubewano = 16
	ExSDO      = 17 // Scattered disk
)

// An ExportUnmarshallFunc unmarshals a single orbit into a struct.
//
// The argument b is the orbit to unmarshal.
//
// ExportUnmarshallFuncs are created with NewExportUnmarshaler.
// The result of a call to the ExportUnmarshallFunc is left in the struct
// that was specified in the call to NewExportUnmarshaler.
type ExportUnmarshallFunc func(b []byte) error

type fieldFunc func([]byte) error

// NewExportUnmarshaler returns a function that will unmarshal orbits to
// a struct.
//
// The argument v specifies the struct.  The concrete type of v must be
// pointer to struct.
func NewExportUnmarshaler(v interface{}) (ExportUnmarshallFunc, error) {
	if v == nil {
		return nil, errors.New("pointer to struct required")
	}
	vp := reflect.ValueOf(v)
	if vp.Kind() != reflect.Ptr {
		return nil, errors.New("pointer to struct required")
	}
	ve := vp.Elem()
	if ve.Kind() != reflect.Struct {
		return nil, errors.New("pointer to struct required")
	}
	vt := ve.Type()
	fieldFuncs := make([]fieldFunc, ve.NumField())
	var nFields int
	for i := range fieldFuncs {
		fv := ve.Field(i) // settable field Value
		sf := vt.Field(i) // StructField type information
		// read tag key "export", set tfName if found
		var tfName string
		var dd decodeData
		var ok bool
		if tv := sf.Tag.Get("export"); tv > "" {
			if tv == "-" || len(tv) > 1 && tv[:2] == "-," {
				continue
			}
			if dd, ok = tFieldMap[tv]; !ok {
				return nil, errors.New("export tag invalid, field: " + sf.Name)
			}
			tfName = tv
		} else {
			if dd, ok = tFieldMap[sf.Name]; !ok {
				return nil, errors.New("unrecognized field: " + sf.Name)
			}
			tfName = sf.Name
		}
		var signed bool
		switch fv.Kind() {
		case reflect.String:
			fieldFuncs[nFields] = strFunc(fv, dd, tfName)
			nFields++
			continue
		case reflect.Int,
			reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			signed = true
			fallthrough
		case reflect.Uint,
			reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if dd.terp != terpInt {
				break // error invalid type
			}
			fieldFuncs[nFields] = intFunc(fv, dd, tfName, sf.Name, signed)
			nFields++
			continue
		case reflect.Float32, reflect.Float64:
			if dd.terp != terpFloat && dd.terp != terpInt {
				break
			}
			var err error
			fieldFuncs[nFields], err = floatFunc(fv, dd, &sf)
			if err != nil {
				return nil, err
			}
			nFields++
			continue
		case reflect.Bool:
			if dd.terp != terpBool {
				break
			}
			fieldFuncs[nFields] = boolFunc(fv, dd, tfName)
			nFields++
			continue
		}
		return nil, errors.New("invald type for field: " + sf.Name)
	}

	// close on fieldFuncs, that's all
	fieldFuncs = fieldFuncs[:nFields]
	return func(data []byte) (err error) {
		for _, f := range fieldFuncs {
			if err = f(data); err != nil {
				return
			}
		}
		return
	}, nil
}

// any field can be requested as string.  for most fields, this means the
// raw text from the field of the text representation.  An exception is
// PlEph, which is expanded into a more readable string.
func strFunc(fv reflect.Value, dd decodeData, tfName string) fieldFunc {
	if tfName == "PlEph" {
		return func(data []byte) error {
			var s string
			switch data[dd.start] {
			case ' ', 'd':
				s = "JPL DE200"
			case 'f':
				s = "JPL DE245"
			case 'h':
				s = "JPL DE403"
			case 'j':
				s = "JPL DE405"
			}
			fv.SetString(s)
			return nil
		}
	}
	return func(data []byte) error {
		fv.SetString(string(bytes.TrimSpace(data[dd.start:dd.end])))
		return nil
	}
}

func intFunc(fv reflect.Value, dd decodeData,
	tfName, sfName string, signed bool) fieldFunc {
	set := reflect.Value.SetUint
	if signed {
		set = func(fv reflect.Value, i uint64) {
			fv.SetInt(int64(i))
		}
	}
	switch tfName {
	case "Precise":
		return func(data []byte) error {
			fs := string(bytes.TrimSpace(data[dd.start:dd.end]))
			i, err := strconv.ParseUint(fs, 16, 64)
			if err != nil {
				return fmt.Errorf("%v. field: %s", err, sfName)
			}
			set(fv, i)
			return nil
		}
	case "YFirst", "YLast":
		return func(data []byte) error {
			fs := string(bytes.TrimSpace(data[dd.start:dd.end]))
			sOpp := string(bytes.TrimSpace(data[123:126]))
			nOpp, err := strconv.ParseUint(sOpp, 10, 64)
			if err != nil {
				return fmt.Errorf("%v. field: NObs", err)
			}
			var i uint64
			if nOpp > 1 {
				i, err = strconv.ParseUint(fs, 10, 64)
				if err != nil {
					return fmt.Errorf("%v. field: %s", err, sfName)
				}
			}
			set(fv, i)
			return nil
		}
	case "Arc":
		return func(data []byte) error {
			fs := string(bytes.TrimSpace(data[dd.start:dd.end]))
			sOpp := string(bytes.TrimSpace(data[123:126]))
			nOpp, err := strconv.ParseUint(sOpp, 10, 64)
			if err != nil {
				return fmt.Errorf("%v. field: NObs", err)
			}
			var i uint64
			if nOpp == 1 {
				i, err = strconv.ParseUint(fs, 10, 64)
				if err != nil {
					return fmt.Errorf("%v. field: %s", err, sfName)
				}
			}
			set(fv, i)
			return nil
		}
	}
	return func(data []byte) error {
		fs := string(bytes.TrimSpace(data[dd.start:dd.end]))
		i, err := strconv.ParseUint(fs, 10, 64)
		if err != nil {
			return fmt.Errorf("%v. field: %s", err, sfName)
		}
		set(fv, i)
		return nil
	}
}

func floatFunc(fv reflect.Value, dd decodeData,
	sf *reflect.StructField) (fieldFunc, error) {
	cf := 1.
	defaultVal := 0.
	useDefault := false
	for _, tag := range strings.Split(sf.Tag.Get("val"), ",") {
		switch tag {
		case "", "deg":
		case "rad":
			cf = math.Pi / 180
		case "defNaN":
			defaultVal = math.NaN()
			useDefault = true
		default:
			return nil, fmt.Errorf("invalid tag: %s field: %s", tag, sf.Name)
		}
	}
	return func(data []byte) error {
		fs := string(bytes.TrimSpace(data[dd.start:dd.end]))
		if z, err := strconv.ParseFloat(fs, 64); err == nil {
			fv.SetFloat(z * cf)
		} else {
			if !useDefault {
				return fmt.Errorf("%v. field: %s", err, sf.Name)
			}
			fv.SetFloat(defaultVal)
		}
		return nil
	}, nil
}

func boolFunc(fv reflect.Value, dd decodeData, tfName string) fieldFunc {
	switch tfName {
	case "EAsm":
		return func(data []byte) error {
			fv.SetBool(data[dd.start] == 'E')
			return nil
		}
	case "DD":
		return func(data []byte) error {
			fv.SetBool(data[dd.start] == 'D')
			return nil
		}
	case "NEO":
		return func(data []byte) error {
			fv.SetBool(data[dd.start]&1<<11 != 0)
			return nil
		}
	case "Km":
		return func(data []byte) error {
			fv.SetBool(data[dd.start]&1<<12 != 0)
			return nil
		}
	case "Seen":
		return func(data []byte) error {
			fv.SetBool(data[dd.start]&1<<13 != 0)
			return nil
		}
	case "Crit":
		return func(data []byte) error {
			fv.SetBool(data[dd.start]&1<<14 != 0)
			return nil
		}
	case "PHA":
		return func(data []byte) error {
			fv.SetBool(data[dd.start]&1<<15 != 0)
			return nil
		}
	}
	panic("boolFunc missing case")
}

/*
func parseEpoch(s string) uint64, error {
	if len(s) < 5 {
		goto fail
	}
	c := s[0]-'A'
	if c > 25 {
		goto fail
	}
	yy, err := strconv.ParseUInt(s[1:2], 10, 64)
	if err != nil {
		goto fail
	}
	var m, d uint8
	switch {
	case s[3] >= '1' && s[3] <= '9':
		m = s[3]-'1'
	case s[3] >= 'A' && s[3] <= 'C':
		m = s[3]-'A'
	default:
		goto fail
	}
	switch {
	case s[4] >= '1' && s[4] <= '9':
		d = s[4]-'1'
	case s[4] >= 'A' && s[4] <= 'V':
		d = s[3]-'A'
	default:
		goto fail
	}
	return (c+1)
	}
fail:
	return 0, errors.New("
}
*/
