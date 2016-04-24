// Copyright 2016 Jacob Taylor jacob.taylor@gmail.com
// License: Apache2
package utils

import (
    "fmt"
    "os"
    "encoding/binary"
)


const (
    NBD_REQUEST_MAGIC =                 0x25609513
    NBD_REPLY_MAGIC =                   0x67446698
    NBD_SERVER_SEND_REPLY_MAGIC =       uint64(0x3e889045565a9)
)

func ErrorCheck(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encountered: %v", err)
		os.Exit(0)
	}
}

func LogData(msg string, count int, data []byte) {
    fmt.Printf("%5s (count %3d) Data: '%s' (%v)\n", msg, count, string(data[0:count]), data[0:count])
}

func EncodeInt(val int) []byte {
    data := make([]byte, 4)
    binary.BigEndian.PutUint32(data, uint32(val))
    return data
}

