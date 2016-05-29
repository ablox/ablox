// Copyright 2016 Jacob Taylor jacob.taylor@gmail.com
// License: Apache2
package main

import (
    "fmt"
    "net"
    "../utils"
    "bufio"
    "encoding/binary"
    //"time"
    "os"
    "bytes"
    "io"
    "io/ioutil"
    "log"
)

const nbd_folder = "/sample_disks/"

var characters_per_line = 100
var newline = 0
var line_number = 0

func send_export_list_item(output *bufio.Writer, options uint32, export_name string) {
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
    send_message(output, options, reply_type, uint32(offset), data)
}

func send_ack(output *bufio.Writer, options uint32) {
    send_message(output, options, utils.NBD_COMMAND_ACK, 0, nil)
}

func export_name(output *bufio.Writer, conn net.Conn, payload_size int, payload []byte, options uint32) {
    fmt.Printf("have request to bind to: %s\n", string(payload[:payload_size]))

    defer conn.Close()

    var filename bytes.Buffer
    current_directory, err := os.Getwd()
    utils.ErrorCheck(err)
    filename.WriteString(current_directory)
    filename.WriteString(nbd_folder)
    filename.Write(payload[:payload_size])

    fmt.Printf("Opening file: %s\n", filename.String())

    file, err := os.OpenFile(filename.String(), os.O_RDWR, 0644)

    utils.ErrorCheck(err)
    if err != nil {
        return
    }

    buffer := make([]byte, 256)
    offset := 0

    fs, err := file.Stat()
    file_size := uint64(fs.Size())

    binary.BigEndian.PutUint64(buffer[offset:], file_size)  // size
    offset += 8

    binary.BigEndian.PutUint16(buffer[offset:], 1)  // flags
    offset += 2

    if (options & utils.NBD_FLAG_NO_ZEROES) != utils.NBD_FLAG_NO_ZEROES {
        offset += 124               // pad with 124 zeroes
    }

    _, err = output.Write(buffer[:offset])
    output.Flush()
    utils.ErrorCheck(err)

    buffer = make([]byte, 2048*1024)    // set the buffer to 2mb
    conn_reader := bufio.NewReader(conn)
    for {
        waiting_for := 28       // wait for at least the minimum payload size

        _, err := io.ReadFull(conn_reader, buffer[:waiting_for])
        if err == io.EOF {
            fmt.Printf("Abort detected, escaping processing loop\n")
            break
        }
        utils.ErrorCheck(err)

        //magic := binary.BigEndian.Uint32(buffer)
        command := binary.BigEndian.Uint32(buffer[4:8])
        //handle := binary.BigEndian.Uint64(buffer[8:16])
        from := binary.BigEndian.Uint64(buffer[16:24])
        length := binary.BigEndian.Uint32(buffer[24:28])

        newline += 1;
        if newline % characters_per_line == 0 {
            line_number++
            fmt.Printf("\n%5d: ", line_number * 100)
            newline -= characters_per_line
        }

        switch command {
        case utils.NBD_COMMAND_READ:
            fmt.Printf(".")

            _, err = file.ReadAt(buffer[16:16+length], int64(from))
            utils.ErrorCheck(err)

            binary.BigEndian.PutUint32(buffer[:4], utils.NBD_REPLY_MAGIC)
            binary.BigEndian.PutUint32(buffer[4:8], 0)                      // error bits

            conn.Write(buffer[:16+length])

            continue
        case utils.NBD_COMMAND_WRITE:
            fmt.Printf("W")

            //waiting_for += int(length)                   // wait for the additional payload
            //fmt.Printf("About to read the data that we should be writing out.")
            _, err := io.ReadFull(conn_reader, buffer[28:28+length])
            if err == io.EOF {
                fmt.Printf("Abort detected, escaping processing loop\n")
                break
            }
            utils.ErrorCheck(err)
            //fmt.Printf("Done reading the data that should be written")

            _, err = file.WriteAt(buffer[28:28+length], int64(from))
            utils.ErrorCheck(err)

            file.Sync()

            // let them know we are done
            binary.BigEndian.PutUint32(buffer[:4], utils.NBD_REPLY_MAGIC)
            binary.BigEndian.PutUint32(buffer[4:8], 0)                      // error bits

            conn.Write(buffer[:16])

            continue

        case utils.NBD_COMMAND_DISCONNECT:
            fmt.Printf("D")

            file.Sync()
            return
        }
    }
}

func send_export_list(output *bufio.Writer, options uint32) {
    current_directory, err := os.Getwd()
    files, err := ioutil.ReadDir(current_directory + nbd_folder)
    if err != nil {
        log.Fatal(err)
    }
    for _, file := range files {
        send_export_list_item(output, options, file.Name())
    }

    send_ack(output, options)
}

func send_message(output *bufio.Writer, options uint32, reply_type uint32, length uint32, data []byte ) {
    endian := binary.BigEndian
    buffer := make([]byte, 1024)
    offset := 0

    endian.PutUint64(buffer[offset:], utils.NBD_SERVER_SEND_REPLY_MAGIC)
    offset += 8

    endian.PutUint32(buffer[offset:], options)  // put out the server options
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

var defaultOptions = []byte{0, 0}

func main() {

    if len(os.Args) <  3 {
        panic("missing arguments:  (ipaddress) (portnumber)")
        return
    }

    listener, err := net.Listen("tcp", os.Args[1] + ":" + os.Args[2])
    utils.ErrorCheck(err)

    fmt.Printf("aBlox server online\n")

    reply_magic := make([]byte, 4)
    binary.BigEndian.PutUint32(reply_magic, utils.NBD_REPLY_MAGIC)

    defer fmt.Printf("End of line\n")

    for {
        conn, err := listener.Accept()
        utils.ErrorCheck(err)

        fmt.Printf("We have a new connection from: %s\n", conn.RemoteAddr())
        output := bufio.NewWriter(conn)

        output.WriteString("NBDMAGIC")      // init password
        output.WriteString("IHAVEOPT")      // Magic

        // S: handshake flags
        //output.Write([]byte{0, 3})          // Ubuntu
        output.Write(defaultOptions)        // Qemu
        // END: S: handshake flags

        output.Flush()

        // Fetch the data until we get the initial options
        data := make([]byte, 1024)
        offset := 0
        waiting_for := 16       // wait for at least the minimum payload size

        _, err = io.ReadFull(conn, data[:waiting_for])
        utils.ErrorCheck(err)

        options := binary.BigEndian.Uint32(data[:4])
        command := binary.BigEndian.Uint32(data[12:16])

        // If we are requesting an export, make sure we have the length of the data for the export name.
        if binary.BigEndian.Uint32(data[12:]) == utils.NBD_COMMAND_EXPORT_NAME {
            waiting_for += 4
            _, err = io.ReadFull(conn, data[16:20])
            utils.ErrorCheck(err)
        }
        payload_size := int(binary.BigEndian.Uint32(data[16:]))

        fmt.Sprintf("command is: %d\npayload_size is: %d\n", command, payload_size)
        offset = waiting_for
        waiting_for += int(payload_size)
        _, err = io.ReadFull(conn, data[offset:waiting_for])
        utils.ErrorCheck(err)

        payload := make([]byte, payload_size)
        if payload_size > 0 {
            copy(payload, data[20:])
        }

        utils.LogData("Payload is:", payload_size, payload)
        fmt.Printf("command is: %v\n", command)

        // At this point, we have the command, payload size, and payload.
        switch command {
        case utils.NBD_COMMAND_LIST:
            send_export_list(output, options)
            conn.Close()
            break
        case utils.NBD_COMMAND_EXPORT_NAME:
            go export_name(output, conn, payload_size, payload, options)
            break
        }
    }

}
