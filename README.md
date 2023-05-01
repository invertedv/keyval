## package keyval

[![Go Report Card](https://goreportcard.com/badge/github.com/invertedv/keyval)](https://goreportcard.com/report/github.com/invertedv/keyval)
[![godoc](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/mod/github.com/invertedv/keyval?tab=overview)

Package keyval provides a convenient method handling data in a key/value format.

### Features of the keyval package

The package revolves around the KeyValue map which maps the keys to the values. The map can be created by 
reading the keyvals from a file or from two slices of strings, one being keys the other values. 
The file format has the form:

    <key>: <value(s)>
When reading from a file, values can cross multiple lines in the file. Both inline and standalone 
comments in the keyval file are supported. Comments use the Go // syntax.

Values are stored in a struct that converts the value(s) into all the types the value supports. These can be:

- string
- int
- float64
- date (time.Time)
- []string
- []int
- []float64
- []time.Time

The struct includes a BestType field that is the "best" type the value can be. The order of precedence, in decreasing order, is:

- date (time.Time)
- int
- float64
- string

Note that slices take precedence over unary types.

Duplicate keys are allowed. If duplicates are detected, a "count" is appended to the key, starting with "1". Duplicates are numbered in the order they are found in the file. The above can cause problems if you intend to have "key", "key" *and* another key called "key1" -- so beware.

If the value can be parsed as a slice, leading and trailing spaces are removed after the string is split into a slice. The default delimiter for slices is ",". If you have dates like "January 2, 2000", you'll need to change it to something else.

There is one special key: include. The value associated with this key is a file name. The kevvals from the specified file are loaded when the "include" key is encountered.

There are functions to check whether required keys are present and whether extra keys are present. There is also a validation function: CheckLegals. See the example.

Date formats that are accepted are:

    "20060102"
    "01/02/2006"
    "1/2/2006"
    "January 2, 2006"
    "Jan 2, 2006"
