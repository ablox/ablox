// Copyright 2016 Jacob Taylor jacob.taylor@gmail.com
// License: Apache2
package utils

import (
    "fmt"
    "os"
)

func ErrorCheck(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encountered: %v", err)
		os.Exit(0)
	}
}
