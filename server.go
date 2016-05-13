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
    "os"
    "bytes"
    "io"
    "io/ioutil"
    "os/user"
    "log"
)

const nbd_folder = "/sample_disks/"

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

func get_user_home_dir() (homedir string) {
    usr, err := user.Current()
    if err != nil {
        log.Fatal(err)
    }
    return usr.HomeDir
}

func export_name(output *bufio.Writer, conn net.Conn, payload_size int, payload []byte) {
    fmt.Printf("have request to bind to: %s\n", string(payload[:payload_size]))

    defer conn.Close()

    var filename bytes.Buffer
    filename.WriteString(get_user_home_dir() + nbd_folder)

    actual_filename := string(payload[:payload_size])

    if actual_filename == "export" {
        actual_filename = "test_fake_hdd"
    }

    filename.WriteString(actual_filename)
    fmt.Printf("Opening file: %s", filename.String())

    // attempt to open the file read only
    file, err := os.Open(filename.String())
    utils.ErrorCheck(err)

    buffer := make([]byte, 256)
    offset := 0

    fs, err := file.Stat()
    file_size := uint64(fs.Size())

    binary.BigEndian.PutUint64(buffer[offset:], file_size)  // size
    offset += 8

    binary.BigEndian.PutUint16(buffer[offset:], 1)  // flags
    offset += 2

    data_out, err := output.Write(buffer[:offset])
    output.Flush()
    utils.ErrorCheck(err)
    fmt.Printf("Wrote %d chars: %v\n", data_out, buffer[:offset])

    buffer = make([]byte, 512*1024)
    file_position := uint64(0)
    conn_reader := bufio.NewReader(conn)
    abort := false
    for {

        offset := 0
        waiting_for := 28       // wait for at least the minimum payload size

        for offset < waiting_for {
            length, err := conn_reader.Read(buffer[offset:waiting_for])
            offset += length
            utils.ErrorCheck(err)
            if err == io.EOF {
                abort = true
                break
            }
            utils.LogData("Reading instruction\n", offset, buffer)
            if offset < waiting_for {
                time.Sleep(5 * time.Millisecond)
            }
        }
        if abort {
            fmt.Printf("Abort detected, escaping processing loop\n")
            break
        }

        fmt.Printf("We read the buffer %v\n", buffer[:waiting_for])

        //magic := binary.BigEndian.Uint32(buffer)
        command := binary.BigEndian.Uint32(buffer[4:8])
        handle := binary.BigEndian.Uint64(buffer[8:16])
        from := binary.BigEndian.Uint64(buffer[16:24])
        length := binary.BigEndian.Uint32(buffer[24:28])

        switch command {
        case utils.NBD_COMMAND_READ:
            fmt.Printf("We have a request to read handle: %v, from: %v, length: %v, file_position: %v\n", handle, from, length, file_position)
            fmt.Printf("Read Resquest    Offset:%x length: %v     Handle %X\n", from, length, handle)
            if file_position != from {
                fmt.Printf("Seeking to %v\n", int64(from))
                file.Seek(int64(from), 0)     // seek to the requested position relative to the start of the file
                file_position = from
            }
            data_out, err = file.Read(buffer[16:16+length])
            file_position += uint64(length)
            fmt.Printf("new file position is: %v\n", file_position)
            utils.ErrorCheck(err)

            // Should not be big indian?
            binary.BigEndian.PutUint32(buffer[:4], utils.NBD_REPLY_MAGIC)
            binary.BigEndian.PutUint32(buffer[4:8], 0)                      // error bits

            utils.LogData("About to reply with", int(16+length), buffer)
            fmt.Printf("length of buffer: %v\n", len(buffer[:16+length]))
            fmt.Printf("tail of buffer: %v\n", buffer[length:16+length])

            conn.Write(buffer[:16+length])

            continue
        case utils.NBD_COMMAND_WRITE:
            fmt.Printf("We have a request to write handle: %v, from: %v, length: %v\n", handle, from, length)
            continue
        case utils.NBD_COMMAND_DISCONNECT:
            fmt.Printf("We have received a request to disconnect\n")
            // close the file and return
            return
        }
    }
}

func send_export_list(output *bufio.Writer) {
    files, err := ioutil.ReadDir(get_user_home_dir() + nbd_folder)
    if err != nil {
        log.Fatal(err)
    }
    for _, file := range files {
        send_export_list_item(output, file.Name())
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

    if len(os.Args) <  3 {
        panic("missing arguments:  (ipaddress) (portnumber)")
        return
    }

    listener, err := net.Listen("tcp", os.Args[1] + ":" + os.Args[2])
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
            conn.Close()
            break
        case utils.NBD_COMMAND_EXPORT_NAME:
            go export_name(output, conn, payload_size, payload)
            break
        }
    }

}
