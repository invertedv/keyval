// Package keyval provides a convenient method handling data in a key/value format.
//
// # Features of the keyval package
//
// The package revolves around the Keyval map which maps the keys to the values. The map can be created by
// reading the keyvals from a file or from two slices of strings, one being keys the other values.
// The file format has the form:
//
//	<key>: <value(s)>
//
// When reading from a file, values can cross multiple lines in the file.
// Both inline and standalone comments in the keyval file are supported. Comments use the Go // syntax.
//
// Values are stored in a struct that converts the value(s) into all the types the value supports. These can be:
//   - string
//   - int
//   - float64
//   - date (time.Time)
//   - []string
//   - []int
//   - []float64
//   - []time.Time
//
// The struct includes a BestType field that is the "best" type
// the value can be.  The order of precedence, in decreasing order, is:
//   - date (time.Time)
//   - int
//   - float64
//   - string
//
// Note that slices take precedence over unary types.
//
// Duplicate keys are allowed. If duplicates are detected, a "count" is appended to the key, starting with "1".
// Duplicates are numbered in the order they are found in the file.
// The above can cause problems if you intend to have "key", "key" *and* another key called "key1" -- so beware.
//
// If the value can be parsed as a slice, leading and trailing spaces are removed after the string is split into a slice.
// The default delimiter for slices is ",". If you have dates like "January 2, 2000", you'll need to change it
// to something else.
//
// There is one special key: include.  The value associated with this key is a file name.  The kevvals from
// the specified file are loaded when the "include" key is encountered.
//
// There are functions to check whether required keys are present and whether extra keys are present.
// There is also a validation function: CheckLegals.  See the example.
//
// Date formats that are accepted are:
//
//	"20060102"
//	"01/02/2006"
//	"1/2/2006"
//	"January 2, 2006"
//	"Jan 2, 2006"
package keyval

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	KVDelim   = ":"  // KVDelim is the delimiter that separates the key from the value
	ListDelim = ","  // ListDelim separates list (slice) elements in the value.
	LineEOL   = "\n" // FileEOF is the end-of-line character
)

// DataType is used to identify the "best" data type of the value.  The decreasing order of precedence is:
//   - slices
//   - unary types
//
// Within each of these types, the order is:
//   - int
//   - float
//   - string
type DataType int

const (
	String DataType = 0 + iota
	Float
	Int
	Date
	SliceStr
	SliceFloat
	SliceInt
	SliceDate
	InValid
)

//go:generate stringer -type=DataType

// The Value struct holds the val part of the keyval.  All legal elements are populated.
type Value struct {
	AsString string
	AsInt    *int
	AsFloat  *float64
	AsDate   *time.Time
	AsSliceS []string
	AsSliceI []int
	AsSliceF []float64
	AsSliceD []time.Time
	BestType DataType
}

// KeyVal holds the map representation of the keyval file.
type KeyVal map[string]*Value

// Get returns a value. Nil is returned if the "want" DataType is not a legal type.
func (kv KeyVal) Get(key string) *Value {
	val, ok := kv[key]

	if !ok {
		return nil
	}

	return val
}

// GetBest returns the Value element of the BestType along with what that type is.
func (kv KeyVal) GetBest(key string) (data any, datatype DataType) {
	val, ok := kv[key]

	if !ok {
		return nil, InValid
	}

	switch val.BestType {
	case String:
		return val.AsString, String
	case Float:
		return val.AsFloat, Float
	case Int:
		return val.AsInt, Int
	case Date:
		return val.AsDate, Date
	case SliceStr:
		return val.AsSliceS, SliceStr
	case SliceFloat:
		return val.AsSliceF, SliceFloat
	case SliceInt:
		return val.AsSliceI, SliceInt
	case SliceDate:
		return val.AsSliceD, SliceDate
	}

	return nil, InValid
}

// GetMultiple retrieves all the Values that start with root that have duplicate keys. The actual keys would be
// "root"1, "root"2, ....  The keys are returned in order.
func (kv KeyVal) GetMultiple(root string) []*Value {
	val := kv[root+"1"]
	if val == nil {
		if val = kv[root]; val != nil {
			return []*Value{val}
		}

		return nil
	}

	vals := []*Value{val}
	ind := 2

	for {
		val = kv[fmt.Sprintf("%s%d", root, ind)]
		if val == nil {
			return vals
		}
		vals = append(vals, val)
		ind++
	}
}

// Missing returns a slice of needles that are not keys in kv.
// needles is a comma-separated list of keys to look for.
// returns nil if all needles are present.
func (kv KeyVal) Missing(needles string) (missing []string) {
	if needles == "" {
		return nil
	}

	needles = CleanString(needles, " \n\t")
	for _, miss := range strings.Split(needles, ",") {
		if kv.Get(miss) == nil && kv.GetMultiple(miss) == nil {
			missing = append(missing, miss)
		}
	}

	return missing
}

// Present returns the keys in needles that are in kv.
func (kv KeyVal) Present(needles string) (present []string) {
	if needles == "" {
		return nil
	}

	needles = CleanString(needles, " \n\t")
	for _, ndl := range strings.Split(needles, ListDelim) {
		if kv.Get(ndl) != nil {
			present = append(present, ndl)
		}
	}

	return present
}

// Unknown returns the keys in kv that are not in universe.
// universe is a comma-separated string that has the universe of known keys.
// returns nil if all keys in kv are in universe.
// Any entry in universe that ends in * is treated as a wildcard
func (kv KeyVal) Unknown(universe string) (novel []string) {
	if universe == "" {
		return nil
	}

	// remove potential dreck
	universe = CleanString(universe, " \n\t")

	univSlc := strings.Split(universe, ",")
	for key := range kv {
		found := false

		for _, uni := range univSlc {
			if uni == key {
				found = true
				break
			}

			if uni[len(uni)-1] == '*' {
				shortUni := uni[:len(uni)-1]
				if len(key) >= len(shortUni) && shortUni == key[:len(shortUni)] {
					found = true
					break
				}
			}
		}
		if !found {
			novel = append(novel, key)
		}
	}

	return novel
}

// ReadKV2Slc reads the specFile and returns the key/vals as two slices of strings.
// These can be processed into a KeyVal by ProcessKVs.
func ReadKV2Slc(specFile string) (keys, vals []string, err error) {
	handle, e := os.Open(specFile)
	if e != nil {
		return nil, nil, e
	}
	defer func() { _ = handle.Close() }()

	rdr := bufio.NewReader(handle)

	// must keep track of multiple lines since values can occupy multiple lines.
	line, nextLine := "", ""
	done := 0 // done==2: processing ends; done==1: hit EOF, but it occurs on a populated line so will do 1 more loop.

	for {
		nextLine = line

		for done == 0 {
			if line, e = rdr.ReadString(LineEOL[0]); e == io.EOF {
				done = 1 // hit EOF, so process nextLine and line
				if line == "" {
					done = 2 // EOF and the line was blank--so process nextline and quit
				}
				break
			}

			// hit an actual error
			if e != nil && e != io.EOF {
				return nil, nil, e
			}

			line = strings.TrimLeft(strings.TrimRight(line, LineEOL), " ")

			// lines must be at least 2 characters
			if line == "" || len(line) < 2 {
				continue
			}

			// entire line is a comment
			if line[0:2] == "//" {
				continue
			}

			// line has comment
			if ind := strings.Index(line, "//"); ind >= 0 {
				line = line[0:ind]
				line = strings.TrimRight(line, " ")
			}

			// are these separate entries?
			if strings.Contains(nextLine, KVDelim) && strings.Contains(line, KVDelim) {
				break
			}

			// append and keep reading
			nextLine = fmt.Sprintf("%s %s", nextLine, line)
		}

		// split into key and val
		kvSlice := strings.SplitN(nextLine, KVDelim, 2)
		if len(kvSlice) != 2 {
			return nil, nil, fmt.Errorf("bad key val: %s in file %s", nextLine, specFile)
		}

		key := strings.ReplaceAll(kvSlice[0], " ", "")
		val := strings.TrimLeft(kvSlice[1], " ")
		if key == "include" {
			ks, vs, e := ReadKV2Slc(val)
			if e != nil {
				return nil, nil, e
			}

			for ind := 0; ind < len(ks); ind++ {
				keys = append(keys, ks[ind])
				vals = append(vals, vs[ind])
			}

			continue
		}

		keys = append(keys, key)
		vals = append(vals, val)

		if done == 2 {
			return keys, vals, nil
		}

		// The next iteration will be the last.  We won't do any more reading if done=1.
		if done == 1 {
			done++
		}
	}
}

// ProcessKVs process keys and vals as two slices of string.  It returns a KeyVal.
func ProcessKVs(keys, vals []string) (kv KeyVal, err error) {
	if keys == nil || vals == nil {
		return nil, fmt.Errorf("nil slice passes to ProcessKVs")
	}

	if len(keys) != len(vals) {
		return nil, fmt.Errorf("slices not same length in ProcessKVs")
	}

	kv = make(KeyVal)
	for indx := 0; indx < len(keys); indx++ {
		// spaces mean nothing
		base := keys[indx]

		// now we test to see if this key is a duplicate
		key, keyTest := base, base

		// if key isn't there but if it's a duplicate, the first entry might already have had "1" appended.
		if _, ok := kv[base]; !ok {
			keyTest = base + "1"
		}

		// look for duplicates and stop when we run out
		ind := 1
		for _, ok := kv[keyTest]; ok; _, ok = kv[keyTest] {
			ind++
			keyTest = fmt.Sprintf("%s%d", base, ind)
			key = keyTest
		}

		// In this case, we have a duplicate but this is the first dup.  In that case, append a "1" to the first
		// instance and drop the original.
		if ind == 2 {
			kv[base+"1"] = kv[base]
			delete(kv, base)
		}

		kv[key] = Populate(vals[indx])
	}

	return kv, nil
}

// ReadKV reads a key/val set from specFile and returns KeyVal
func ReadKV(specFile string) (keyval KeyVal, err error) {
	keys, vals, e := ReadKV2Slc(specFile)
	if e != nil {
		return keyval, e
	}

	return ProcessKVs(keys, vals)
}

// toDate attempts to convert inStr to time.Time
func toDate(inStr string) *time.Time {
	fmts := []string{"2006-01-02", "2006-1-2", "2006/01/02", "2006/1/2", "20060102", "01022006",
		"01/02/2006", "1/2/2006", "01-02-2006", "1-2-2006", "200601", "Jan 2 2006", "January 2 2006",
		"Jan 2, 2006", "January 2, 2006", time.RFC3339}
	trim := strings.TrimRight(strings.TrimLeft(inStr, " "), " ")
	for _, fm := range fmts {
		dt, err := time.Parse(fm, trim)
		if err == nil {
			return &dt
		}
	}

	return nil
}

// Populate populates all the legal values that valStr can accommodate.  The AsString field is always populated.
// The BestType is set using the order of precedence described under the type DataType.
func Populate(valStr string) *Value {
	val := &Value{AsString: valStr, BestType: String}

	if valFloat, e := strconv.ParseFloat(strings.ReplaceAll(valStr, " ", ""), 64); e == nil {
		toFloat := valFloat
		val.AsFloat = &toFloat
		val.BestType = Float
	}

	if valInt, e := strconv.ParseInt(strings.ReplaceAll(valStr, " ", ""), 10, 64); e == nil {
		toInt := int(valInt)
		val.AsInt = &toInt
		val.BestType = Int
	}

	if valDt := toDate(valStr); valDt != nil {
		val.AsDate = valDt
		val.BestType = Date
	}

	if slcS, slcI, slcF, slcD := toSlices(valStr); slcS != nil {
		val.AsSliceS, val.AsSliceI, val.AsSliceF, val.AsSliceD = slcS, slcI, slcF, slcD
		if len(slcS) > 1 {
			val.BestType = SliceStr
		}

		// check slice has more than one element to call it the best choice
		if len(slcF) > 1 {
			val.BestType = SliceFloat
		}

		if len(slcI) > 1 {
			val.BestType = SliceInt
		}

		if len(slcD) > 1 {
			val.BestType = SliceDate
		}
	}

	return val
}

// toSlices converts input into all the slice types it supports.
func toSlices(input string) (asStr []string, asInt []int, asFloat []float64, asDate []time.Time) {
	asStr = strings.Split(input, ListDelim)
	// after split, trim off leading/trailing spaces
	for ind, str := range asStr {
		asStr[ind] = strings.TrimRight(strings.TrimLeft(str, " "), " ")
	}

	asInt = make([]int, 0)
	asFloat = make([]float64, 0)
	asDate = make([]time.Time, 0)

	for ind := 0; ind < len(asStr); ind++ {
		if val, e := strconv.ParseInt(strings.ReplaceAll(asStr[ind], " ", ""), 10, 64); e == nil {
			asInt = append(asInt, int(val))
		}
		if val, e := strconv.ParseFloat(strings.ReplaceAll(asStr[ind], " ", ""), 64); e == nil {
			asFloat = append(asFloat, val)
		}

		if val := toDate(asStr[ind]); val != nil {
			asDate = append(asDate, *val)
		}
	}

	if len(asInt) != len(asStr) {
		asInt = nil
	}

	if len(asFloat) != len(asStr) {
		asFloat = nil
	}

	if len(asDate) != len(asStr) {
		asDate = nil
	}

	return asStr, asInt, asFloat, asDate
}

// CleanString removes all the characters in cutSet from str
func CleanString(str, cutSet string) string {
	for ind := 0; ind < len(cutSet); ind++ {
		str = strings.ReplaceAll(str, cutSet[ind:ind+1], "")
	}

	return str
}

// BuildLegals takes the string in legal.txt returning 3 slices. The first is the target key,
// the second is a category and the third is the value.
// The format for the string is:
// key:required-<yes/no>
// key:type-<string/int/float>
// key:multiples-<yes/no>
// key:requires-<another key name>
//
// Only the first two are required.
func BuildLegals(legalKeys string) (keys, field, val []string) {
	for _, lgl := range strings.Split(legalKeys, "\n") {
		if lgl == "" {
			continue
		}

		kv := strings.Split(lgl, ":")
		keys = append(keys, kv[0])
		fv := strings.Split(kv[1], "-")
		field = append(field, fv[0])
		val = append(val, fv[1])
	}

	return keys, field, val
}

// getLgl returns the value from the key/field/value triple in keys/legal.txt
func getLgl(key, field string, kl, fl, vl []string) (val string) {
	for ind := 0; ind < len(kl); ind++ {
		if kl[ind] == key && fl[ind] == field {
			return vl[ind]
		}
	}

	return ""
}

// CheckLegals builds the legal keys, types and "required" then checks kv against this.
// CheckLegals returns the first error it finds in this order:
//   - missing required key
//   - bad value
//   - unknown keys
//
// If you don't care about extra keys, you can just ignore the last error.
func CheckLegals(kv KeyVal, legalKeys string) error {
	kl, fl, vl := BuildLegals(legalKeys)

	// keys that admit duplicates need a * appended to their names
	var unique []string
	for ind, k := range kl {
		if fl[ind] == "required" {
			keyn := k
			if getLgl(k, "multiple", kl, fl, vl) == "yes" {
				keyn += "*"
			}
			unique = append(unique, keyn)
		}
	}

	// required keys
	for ind, k := range kl {
		if fl[ind] == "required" && vl[ind] == "yes" && kv.Missing(k) != nil {
			return fmt.Errorf("missing required key %s", k)
		}
	}

	// cycle through and check types and required secondary keys
	for k, v := range kv {
		if vType := getLgl(k, "type", kl, fl, vl); vType == "int" {
			if v.AsInt == nil {
				return fmt.Errorf("value to key %s must be integer", k)
			}
		}

		// see if there is a list of legal values
		if vals := getLgl(k, "values", kl, fl, vl); vals != "" {
			if searchSlice(v.AsString, strings.Split(vals, ",")) < 0 {
				return fmt.Errorf("illegal value %s for key %s", v.AsString, k)
			}
		}

		// see if another key is required
		if requires := getLgl(k, "requires", kl, fl, vl); requires != "" {
			if kv.Missing(requires) != nil {
				return fmt.Errorf("missing required key %s", requires)
			}
		}
	}

	// look for unrecognized keys
	if unks := kv.Unknown(strings.Join(unique, ",")); unks != nil {
		return fmt.Errorf("unknown key(s): %v", unks)
	}

	return nil
}

// searchSlice checks the joinField is present in the Pipeline
func searchSlice(needle string, haystack []string) (loc int) {
	for ind, hay := range haystack {
		if needle == hay {
			return ind
		}
	}

	return -1
}
