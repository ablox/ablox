// Copyright 2016 Jacob Taylor jacob.taylor@gmail.com
// License: Apache2
package utils

import (
    "fmt"
    "encoding/binary"
    "os"
)


const (
    NBD_REQUEST_MAGIC =                 uint32(0x25609513)
    NBD_REPLY_MAGIC =                   uint32(0x67446698)
    NBD_SERVER_SEND_REPLY_MAGIC =       uint64(0x3e889045565a9)
    NBD_COMMAND_ACK =                   uint32(1)

    NBD_COMMAND_READ =                  uint32(0)
    NBD_COMMAND_WRITE =                 uint32(1)
    NBD_COMMAND_DISCONNECT =            uint32(2)
    NBD_COMMAND_FLUSH =                 uint32(3)
    NBD_COMMAND_TRIM =                  uint32(4)

    NBD_COMMAND_EXPORT_NAME =           uint32(1)
    NBD_COMMAND_LIST =                  uint32(3)
)

func ErrorCheck(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encountered: %v\n", err)
        panic("Nooooooooo\n")
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

