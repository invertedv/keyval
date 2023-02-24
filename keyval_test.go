package keyval

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestKeyVal_Present tests the Present func
func TestKeyVal_Present(t *testing.T) {
	dataPath := os.Getenv("data")
	fileName := dataPath + "/specs1.txt"
	expKey := "a,b,X,Y,d"
	expPresent := []string{"a", "b", "d"}

	var kv KeyVal
	var e error

	if kv, e = ReadKeyVal(fileName); e != nil {
		panic(e)
	}
	missing := kv.Present(expKey)
	assert.ElementsMatch(t, missing, expPresent)

	expKey = "A,X"
	missing = kv.Present(expKey)
	assert.Nil(t, missing)
}

// TestKeyVal_Missing tests the Missing func
func TestKeyVal_Missing(t *testing.T) {
	dataPath := os.Getenv("data")
	fileName := dataPath + "/specs1.txt"
	expKey := "a,b,X,Y,d"
	expMiss := []string{"X", "Y"}

	var kv KeyVal
	var e error

	if kv, e = ReadKeyVal(fileName); e != nil {
		panic(e)
	}
	missing := kv.Missing(expKey)
	assert.ElementsMatch(t, missing, expMiss)

	expKey = "a,b,d"
	missing = kv.Missing(expKey)
	assert.Nil(t, missing)
}

func TestKeyVal_Unknown(t *testing.T) {
	dataPath := os.Getenv("data")
	fileName := dataPath + "/specs1.txt"
	univ := "a,b,c"
	expUnk := []string{"d", "e", "f"}

	var kv KeyVal
	var e error

	if kv, e = ReadKeyVal(fileName); e != nil {
		panic(e)
	}
	unk := kv.Unknown(univ)
	assert.ElementsMatch(t, unk, expUnk)
}

// TestReadKeyVal tests reading a keyval file.
func TestReadKeyVal(t *testing.T) {
	dataPath := os.Getenv("data")
	fileName := dataPath + "/specs1.txt"
	exp := []DataType{String, SliceStr, Int, Float, SliceInt, SliceFloat}
	expKey := []string{"a", "b", "c", "d", "e", "f"}

	var kv KeyVal
	var e error

	if kv, e = ReadKeyVal(fileName); e != nil {
		panic(e)
	}

	for ind, k := range expKey {
		assert.Equal(t, exp[ind], kv[k].BestType)
	}
}

// TestKeyVal_GetMultiple tests (a) multiple keys and (b) EOF on a populated & blank line.
func TestKeyVal_GetMultiple(t *testing.T) {
	dataPath := os.Getenv("data")
	expKey := []string{"eqn1", "eqn2", "eqn3"}

	for ind := 2; ind < 4; ind++ {
		fileName := fmt.Sprintf("%s/specs%d.txt", dataPath, ind)

		var kv KeyVal
		var e error

		kv, e = ReadKeyVal(fileName)
		if e != nil {
			panic(e)
		}

		for _, eq := range expKey {
			assert.NotNil(t, kv[eq])
		}
	}
}

// This example shows the result of reading the specs1.txt file located in the data directory of this package.
func ExampleReadKeyVal() {
	dataPath := os.Getenv("data")
	fileName := dataPath + "/specs1.txt"

	kv, e := ReadKeyVal(fileName)
	if e != nil {
		panic(e)
	}

	choose := []string{"a", "b", "c", "d", "e", "f"}

	for ind := 0; ind < len(choose); ind++ {
		k := choose[ind]
		v := kv[k]
		fmt.Println(k)
		fmt.Println("string: ", v.AsString)
		if v.AsInt != nil {
			fmt.Println("int: ", *v.AsInt)
		}
		if v.AsFloat != nil {
			fmt.Println("float: ", *v.AsFloat)
		}
		if v.AsSliceS != nil {
			fmt.Println("slice: ", v.AsSliceS)
		}
		fmt.Println("best: ", v.BestType)
		fmt.Println()
	}
	// output:
	// a
	// string:  hello
	// best:  String
	//
	// b
	// string:  a,b,c, d,e,f
	// slice:  [a b c d e f]
	// best:  SliceStr
	//
	// c
	// string:  1
	// int:  1
	// float:  1
	// best:  Int
	//
	// d
	// string:  3.2
	// float:  3.2
	// best:  Float
	//
	// e
	// string:  1,2,3,4
	// slice:  [1 2 3 4]
	// best:  SliceInt
	//
	// f
	// string:  1.1, 3,4,5,8.9
	// slice:  [1.1 3 4 5 8.9]
	// best:  SliceFloat
}
