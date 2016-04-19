// Copyright 2016 Jacob Taylor jacob.taylor@gmail.com
// License: Apache2
package main

import (
    "fmt"
    "net"
    "./utils"
    "bufio"
)

func main() {
    controlC := [...]byte{255, 244, 255, 253, 6}
    listener, err := net.Listen("tcp", "192.168.214.1:8000")
    utils.ErrorCheck(err)

    fmt.Printf("Hello World, we have %v\n", listener)

    for {
        conn, err := listener.Accept()
        utils.ErrorCheck(err)

        fmt.Printf("We have a new connection from: %s", conn.RemoteAddr())
        output := bufio.NewWriter(conn)
        output.WriteString("NBDMAGIC")

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
