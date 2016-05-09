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

func export_name(output *bufio.Writer, conn net.Conn, payload_size int, payload []byte) {
    fmt.Printf("have request to bind to: %s\n", string(payload[:payload_size]))

    defer conn.Close()

    var filename bytes.Buffer
    filename.WriteString("/Users/jacob/work/nbd/sample_disks/")
    filename.WriteString(string(payload[:payload_size]))
    fmt.Println("Opening file: %s", filename.String())

    // attempt to open the file read only
    file, err := os.Open(filename.String())
    utils.ErrorCheck(err)
    //defer file.Close()

    data := make([]byte, 1024)
    count, err := file.Read(data)
    utils.ErrorCheck(err)

    if count > 100 {
        count = 100
    }

    // send export information
    // size u64
    // flags u16
    // Zeros (124 bytes)?
    buffer := make([]byte, 256)
    offset := 0

    binary.BigEndian.PutUint64(buffer[offset:], 52428800)  // size
    offset += 8

    binary.BigEndian.PutUint16(buffer[offset:], 0)  // flags
    offset += 2

    //offset += 124       // zero pad

    len, err := output.Write(buffer[:offset])
    output.Flush()
    utils.ErrorCheck(err)
    fmt.Printf("Wrote %d chars: %v\n", len, buffer[:offset])

    fmt.Printf("File descriptor:\n%+v\n", *file)
    fmt.Printf("First 100 bytes: \n%v\n", data[:count])


    // send a reply with the handle
    //S: 32 bits, 0x67446698, magic (NBD_REPLY_MAGIC)
    //S: 32 bits, error (MAY be zero)
    //S: 64 bits, handle
    //S: (length bytes of data if the request is of type NBD_CMD_READ)
    //offset = 0
    //binary.BigEndian.PutUint32(buffer[offset:], utils.NBD_REPLY_MAGIC)
    //offset += 4
    //
    //binary.BigEndian.PutUint32(buffer[offset:], 0) // error
    //offset += 4
    //
    //binary.BigEndian.PutUint64(buffer[offset:], 8000) // handle
    //offset += 8
    //
    //fmt.Printf("Writing out data: %v\n", buffer[:offset])
    //len, err = output.Write(buffer[:offset])
    //output.Flush()
    //utils.ErrorCheck(err)
    fmt.Printf("Done sending data\n")

    buffer = make([]byte, 512*1024)
    file_position := uint64(0)
    conn_reader := bufio.NewReader(conn)
    for {

        offset := 0
        waiting_for := 28       // wait for at least the minimum payload size

        for offset < waiting_for {
            length, err := conn_reader.Read(buffer[offset:waiting_for])
            offset += length
            utils.ErrorCheck(err)
            utils.LogData("Reading instruction", offset, buffer)
            if offset < waiting_for {
                time.Sleep(5 * time.Millisecond)
            }
        }

        //magic := binary.BigEndian.Uint32(buffer)
        command := binary.BigEndian.Uint32(buffer[4:8])
        handle := binary.BigEndian.Uint64(buffer[8:16])
        from := binary.BigEndian.Uint64(buffer[16:24])
        length := binary.BigEndian.Uint32(buffer[24:28])

        switch command {
        case utils.NBD_COMMAND_READ:
            fmt.Printf("We have a request to read handle: %v, from: %v, length: %v, file_position: %v\n", handle, from, length, file_position)
            if file_position != from {
                fmt.Printf("Seeking to %v\n", int64(from))
                file.Seek(int64(from), 0)     // seek to the requested position relative to the start of the file
                file_position = from
            }
            len, err = file.Read(buffer[28:28+length])
            file_position += uint64(length)
            fmt.Printf("new file position is: %v\n", file_position)
            utils.ErrorCheck(err)

            binary.BigEndian.PutUint32(buffer[:4], utils.NBD_REPLY_MAGIC)
            binary.BigEndian.PutUint32(buffer[4:8], 0)                      // error bits

            utils.LogData("About to reply with", int(28+length), buffer)
            conn.Write(buffer[:28+length])

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

    //
    //
    //// Fetch the data until we get the initial options
    //data := make([]byte, 1024)
    //offset := 0
    //waiting_for := 16       // wait for at least the minimum payload size
    //
    //for offset < waiting_for {
    //    length, err := conn.Read(data[offset:])
    //    offset += length
    //    utils.ErrorCheck(err)
    //    utils.LogData("Reading instruction", offset, data)
    //    if offset < waiting_for {
    //        time.Sleep(5 * time.Millisecond)
    //    }
    //}
    //
    //// Skip the first 8 characters (options)
    //command := binary.BigEndian.Uint32(data[12:])
    //payload_size := int(binary.BigEndian.Uint32(data[16:]))
    //
    //
    //    syscall.Read(nbd.socket, buf[0:28])
    //
		//x.magic = binary.BigEndian.Uint32(buf)
		//x.typus = binary.BigEndian.Uint32(buf[4:8])
		//x.handle = binary.BigEndian.Uint64(buf[8:16])
		//x.from = binary.BigEndian.Uint64(buf[16:24])
		//x.len = binary.BigEndian.Uint32(buf[24:28])
    //
    //
    //
    //// Duplicated code. move this to helper function
    //// Duplicated code. move this to helper function
    //// Duplicated code. move this to helper function
    //// Duplicated code. move this to helper function
    //// Fetch the data until we get the initial options
    //time.Sleep(300 * time.Millisecond)
    //
    //fmt.Printf("about to read\n")
    //for ; ;  {
    //    var zero_time time.Time
    //    conn.SetReadDeadline(zero_time)
    //    short_data := make([]byte, 1)
    //    conn.Read(short_data)
    //    fmt.Printf("read byte: %v\n", short_data)
    //    time.Sleep(300 * time.Millisecond)
    //
    //}
    //
    //
    ////conn2, err = listener.Accept()
    ////utils.ErrorCheck(err)
    //
    //data = make([]byte, 1024)
    //offset = 0
    //waiting_for := 16       // wait for at least the minimum payload size
    //
    //for offset < waiting_for {
    //    fmt.Printf("1: offset: %d, data: %v\n", offset, data)
    //    length, err := conn.Read(data[offset:])
    //    offset += length
    //    fmt.Printf("3: offset: %d, err: %v, data: %v\n", offset, err, data)
    //    //utils.ErrorCheck(err)
    //    fmt.Printf("4: offset: %d, data: %v\n", offset, data)
    //    utils.LogData("Reading instruction", offset, data)
    //    fmt.Printf("5: offset: %d, data: %v\n", offset, data)
    //    if offset < waiting_for {
    //    fmt.Printf("6: offset: %d, data: %v\n", offset, data)
    //        time.Sleep(1000 * time.Millisecond)
    //    }
    //}
    fmt.Printf("done reading\n")
    // Duplicated code. move this to helper function
    // Duplicated code. move this to helper function
    // Duplicated code. move this to helper function
    // Duplicated code. move this to helper function



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
            conn.Close()
            break
        case utils.NBD_COMMAND_EXPORT_NAME:
            go export_name(output, conn, payload_size, payload)
            break
        }
    }

}
