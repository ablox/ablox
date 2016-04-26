// Copyright 2016 Jacob Taylor jacob.taylor@gmail.com
// License: Apache2
package main

import (
    "fmt"
    "net"
    "./utils"
    "bufio"
    "encoding/binary"
    "time"
)


func send_export_list_item(output *bufio.Writer, export_name string) {
    data := make([]byte, 1024)
    length := len(export_name)
    offset := 0

    // length of export name
    binary.BigEndian.PutUint32(data[offset:], uint32(length))  // length of string
    offset += 4

    // export name
    copy(data[offset:], export_name)
    offset += length

    reply_type := uint32(2)     // reply_type: NBD_REP_SERVER
    send_message(output, reply_type, uint32(offset), data)
}

func send_ack(output *bufio.Writer) {
    send_message(output, utils.NBD_COMMAND_ACK, 0, nil)
}

func export_name(output *bufio.Writer, payload_size int, payload []byte) {
    fmt.Printf("have request to bind to: %s\n", string(payload[:payload_size]))
}

func send_export_list(output *bufio.Writer) {
    export_name_list := []string{"happy_export", "very_happy_export", "third_export"}

    for index := range export_name_list {
        send_export_list_item(output, export_name_list[index])
    }

    send_ack(output)

}

func send_message(output *bufio.Writer, reply_type uint32, length uint32, data []byte ) {
    endian := binary.BigEndian
    buffer := make([]byte, 1024)
    offset := 0

    endian.PutUint64(buffer[offset:], utils.NBD_SERVER_SEND_REPLY_MAGIC)
    offset += 8

    endian.PutUint32(buffer[offset:], uint32(3))  // Flags (3 = supports list)
    offset += 4

    endian.PutUint32(buffer[offset:], reply_type)  // reply_type: NBD_REP_SERVER
    offset += 4

    endian.PutUint32(buffer[offset:], length)  // length of package
    offset += 4

    if data != nil {
        copy(buffer[offset:], data[0:length])
        offset += int(length)
    }

    data_to_send := buffer[:offset]
    output.Write(data_to_send)
    output.Flush()

    utils.LogData("Just sent:", offset, data_to_send)
}


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

        fmt.Printf("We have a new connection from: %s\n", conn.RemoteAddr())
        output := bufio.NewWriter(conn)

        output.WriteString("NBDMAGIC")      // init password
        output.Flush()
        output.WriteString("IHAVEOPT")      // Magic
        output.Flush()
        output.Write([]byte{0, 3})          // Flags (3 = supports list)
        output.Flush()

        // Fetch the data until we get the initial options
        data := make([]byte, 1024)
        offset := 0
        waiting_for := 16       // wait for at least the minimum payload size

        for offset < waiting_for {
            length, err := conn.Read(data[offset:])
            offset += length
            utils.ErrorCheck(err)
            utils.LogData("Reading instruction", offset, data)
            if offset < waiting_for {
                time.Sleep(5 * time.Millisecond)
            }
        }

        // Skip the first 8 characters (options)
        command := binary.BigEndian.Uint32(data[12:])
        payload_size := int(binary.BigEndian.Uint32(data[16:]))

        fmt.Sprintf("command is: %d\npayload_size is: %d\n", command, payload_size)
        waiting_for += int(payload_size)
        for offset < waiting_for {
            length, err := conn.Read(data[offset:])
            offset += length
            utils.ErrorCheck(err)
            utils.LogData("Reading instruction", offset, data)
            if offset < waiting_for {
                time.Sleep(5 * time.Millisecond)
            }
        }
        payload := make([]byte, payload_size)

        if payload_size > 0{
            copy(payload, data[20:])
        }

        utils.LogData("Payload is:", payload_size, payload)
        fmt.Printf("command is: %v\n", command)

        // At this point, we have the command, payload size, and payload.
        switch command {
        case utils.NBD_COMMAND_LIST:
            send_export_list(output)
            break
        case utils.NBD_COMMAND_EXPORT_NAME:
            export_name(output, payload_size, payload)
            break
        }

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
            fmt.Printf("%s echo: %s '%v'\n", conn.RemoteAddr().String(), input.Text(), input.Bytes())
        }
        conn.Close()

    }

}
