## package keyval

The keyval package provides a convenient method for reading files that have a keyval format.

Features of the keyval file format:

- The file is loaded into a map.
- keyvals can cross multiple lines
- Results are stored in a struct that converts the values into a variety of types:
    - string
    - int
    - float64
    - []string
    - []int
    - []float64<br><br>
- The struct includes a BestType field that is the best type the value can be.  The order
of precedence, in decreasing order, is:
    - int
    - float64
    - string

  Slices take precedence over unary types.<br><br>
- Duplicate keys are allowed. If duplicates are detected, a "count" is appended to the key, starting with "1".
  Duplicates are numbered in the order they are found in the file.
- Both inline and standalone comments in the keyval file are supported. Comments use the Go // syntax.

