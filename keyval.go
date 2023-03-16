// Package keyval provides a convenient method for reading files that have a keyval format.
//
// Features of the keyval file format:
//
// - The file is loaded into a map.
// - keyvals can cross multiple lines.
// - Results are stored in a struct that converts the values into a variety of types:
//   - string
//   - int
//   - float64
//   - []string
//   - []int
//   - []float64
//
// - The struct includes a BestType field that is the best type the value can be.  The order of
// precedence, in decreasing order, is:
//   - int
//   - float64
//   - string
//
// Slices take precedence over unary types.
//
// - Duplicate keys are allowed. If duplicates are detected, a "count" is appended to the key, starting with "1".
// Duplicates are numbered in the order they are found in the file.
//
// - Both inline and standalone comments in the keyval file are supported. Comments use the Go // syntax.
//
// There is one special key: include.  The value associated with this key is a file name.  The kevvals from
// that file are loaded when this key is encountered.
package keyval

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
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
	SliceStr
	SliceFloat
	SliceInt
	InValid
)

//go:generate stringer -type=DataType

// The Value struct holds the val part of the keyval.  All legal elements are populated.
type Value struct {
	AsString string
	AsInt    *int
	AsFloat  *float64
	AsSliceS []string
	AsSliceI []int
	AsSliceF []float64
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
	case SliceStr:
		return val.AsSliceS, SliceStr
	case SliceFloat:
		return val.AsSliceF, SliceFloat
	case SliceInt:
		return val.AsSliceI, SliceInt
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

	for _, miss := range strings.Split(needles, ",") {
		if kv.Get(miss) == nil {
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

	needles = strings.ReplaceAll(strings.ReplaceAll(needles, " ", ""), "\n", "")
	for _, ndl := range strings.Split(needles, ",") {
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
	universe = strings.ReplaceAll(strings.ReplaceAll(universe, " ", ""), "\n", "")

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

// Populate populates all the legal values that valStr can accommodate.  The AsString field is always populated.
// The BestType is set using the order of precedence described under the type DataType.
func Populate(valStr string) *Value {
	val := &Value{AsString: valStr, BestType: String}

	if valFloat, e := strconv.ParseFloat(valStr, 64); e == nil {
		toFloat := valFloat
		val.AsFloat = &toFloat
		val.BestType = Float
	}

	if valInt, e := strconv.ParseInt(valStr, 10, 64); e == nil {
		toInt := int(valInt)
		val.AsInt = &toInt
		val.BestType = Int
	}

	if slcS, slcI, slcF := toSlices(valStr); slcS != nil {
		val.AsSliceS, val.AsSliceI, val.AsSliceF = slcS, slcI, slcF
		val.BestType = SliceStr
		if val.AsSliceF != nil {
			val.BestType = SliceFloat
		}
		if val.AsSliceI != nil {
			val.BestType = SliceInt
		}
	}

	return val
}

// toSlices converts input into all the slice types it supports.
func toSlices(input string) (asStr []string, asInt []int, asFloat []float64) {
	asStr = strings.Split(strings.ReplaceAll(input, " ", ""), ListDelim)

	if len(asStr) > 1 {
		asInt = make([]int, 0)
		asFloat = make([]float64, 0)
		for ind := 0; ind < len(asStr); ind++ {
			if val, e := strconv.ParseInt(asStr[ind], 10, 64); e == nil {
				asInt = append(asInt, int(val))
			}
			if val, e := strconv.ParseFloat(asStr[ind], 64); e == nil {
				asFloat = append(asFloat, val)
			}
		}

		if len(asInt) != len(asStr) {
			asInt = nil
		}

		if len(asFloat) != len(asStr) {
			asFloat = nil
		}

		return asStr, asInt, asFloat
	}

	return nil, nil, nil
}
