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
	KeyValDelim = ":"  // KeyValDelim is the delimiter that separates the key from the value
	ListDelim   = ","  // ListDelim separates list (slice) elements in the value.
	LineEOL     = "\n" // FileEOF is the end-of-line character
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
func (kv KeyVal) Get(key string, want DataType) *Value {
	val, ok := kv[key]

	if !ok {
		return nil
	}

	if want != val.BestType {
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

// ReadKeyVal reads the keyval file and returns the map representation.
// The Value struct is populated with all legitimate representations of value.
// The elements of Value are set for all types the value can be converted to.  The AsString field is always populated.
// The BestType is set using the order of precedence described under the type DataType.
func ReadKeyVal(specFile string) (kv KeyVal, err error) {
	handle, e := os.Open(specFile)
	if e != nil {
		return nil, e
	}
	defer func() { _ = handle.Close() }()

	rdr := bufio.NewReader(handle)
	kv = make(KeyVal)

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
				return nil, e
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
			if strings.Contains(nextLine, KeyValDelim) && strings.Contains(line, KeyValDelim) {
				break
			}

			// append and keep reading
			nextLine = fmt.Sprintf("%s%s", nextLine, line)
		}

		// split into key and val
		kvSlice := strings.SplitN(nextLine, KeyValDelim, 2)
		if len(kvSlice) != 2 {
			return nil, fmt.Errorf("bad key val: %s in file %s", nextLine, specFile)
		}

		// spaces mean nothing
		base := strings.ReplaceAll(kvSlice[0], " ", "")

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

		// fill in the values.
		val := Populate(strings.TrimLeft(kvSlice[1], " "))
		kv[key] = val

		// OK, we are done.
		if done == 2 {
			break
		}

		// The next iteration will be the last.  We won't do any more reading if done=1.
		if done == 1 {
			done++
		}
	}

	return kv, nil
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
