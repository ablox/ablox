// Copyright 2016 Jacob Taylor jacob.taylor@gmail.com
// License: Apache2
package main

import (
    "fmt"
    "net"
    "./utils"
    "bufio"
    "encoding/binary"
)

func main() {
    controlC := [...]byte{255, 244, 255, 253, 6}
    listener, err := net.Listen("tcp", "192.168.214.1:8000")
    utils.ErrorCheck(err)

    fmt.Printf("Hello World, we have %v\n", listener)
    reply_magic := make([]byte, 4)
    binary.BigEndian.PutUint32(reply_magic, utils.NBD_REPLY_MAGIC)

    for {
        conn, err := listener.Accept()
        utils.ErrorCheck(err)

        fmt.Printf("We have a new connection from: %s", conn.RemoteAddr())
        output := bufio.NewWriter(conn)

        output.WriteString("NBDMAGIC")      // init password
        //output.Write(reply_magic)
        output.Flush()
        output.WriteString("IHAVEOPT")      // Magic
        output.Flush()
        output.Write([]byte{0, 3})          // Flags (3 = supports list)
        output.Flush()
        //output.WriteString("\n")

        data := make([]byte, 1024)
        length, err := conn.Read(data)
        utils.ErrorCheck(err)
        utils.LogData("A", length, data)

        data = make([]byte, 1024)
        length, err = conn.Read(data)
        utils.ErrorCheck(err)
        utils.LogData("B", length, data)

        endian := binary.BigEndian


        // Send the first export


        //output.Write(reply_magic)
        data = make([]byte, 1024)
        offset := 0
        endian.PutUint64(data[offset:], utils.NBD_SERVER_SEND_REPLY_MAGIC)
        offset += 8
        //output.WriteString(">EVZ")
        //output.Write(data[0:7])
        //output.Flush()
        //output.WriteString("IHAVEOPT")      // Magic
        //output.Flush()
        endian.PutUint32(data[offset:], uint32(3))  // Flags (3 = supports list)
        offset += 4
        //output.Write([]byte{0, 3})
        //output.Flush()
        // reply type
        endian.PutUint32(data[offset:], uint32(2))  // reply_type: NBD_REP_SERVER
        offset += 4

        //output.Write([]byte{0, 2})          // reply_type: NBD_REP_SERVER
        //output.Flush()

        export_name := "happy_export"
        length = len(export_name)

        // length of export name package
        endian.PutUint32(data[offset:], uint32(length + 4))  // length of string
        offset += 4

        // length of export name
        endian.PutUint32(data[offset:], uint32(length))  // length of string
        offset += 4

        // export name
        copy(data[offset:], export_name)
        offset += length

        output.Write(data)
        output.Flush()

        utils.LogData("Just sent:", offset, data)

        //output.Write(utils.EncodeInt(len(export_name)))
        //// export name
        //output.WriteString(export_name)
        //output.Write([]byte{})
        //output.Flush()

        //data = make([]byte, 1024)
        //length, err = conn.Read(data)
        //utils.ErrorCheck(err)
        //utils.LogData("C", length, data)




        input := bufio.NewScanner(conn)
        for input.Scan() {
            if len(input.Bytes()) == 5 {
                temp := [...]byte{0, 0, 0, 0, 0}
                copy(temp[:], input.Bytes())
                if temp == controlC {
                    fmt.Printf("Control-C received. Bye\n")
                    break
                }
            }
            //fmt.Printf("Error is: %+v", input.Err())
            fmt.Printf("%s echo: %s '%v'\n", conn.RemoteAddr().String(), input.Text(), input.Bytes())
        }
        conn.Close()

    }

}
