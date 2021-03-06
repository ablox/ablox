// Copyright 2016 Jacob Taylor jacob@ablox.io
// License: Apache2 - http://www.apache.org/licenses/LICENSE-2.0
package main

import (
    "fmt"
    "net"
    "../utils"
    "bufio"
    "encoding/binary"
    "os"
    "bytes"
    "io"
    "io/ioutil"
    "github.com/urfave/cli"
    "path/filepath"
    "strconv"
)

const nbd_folder = "/sample_disks/"

var characters_per_line = 100
var newline = 0
var line_number = 0

// settings for the server
type Settings struct {
    ReadOnly    bool
    AutoFlush   bool
    Host        string
    Port        string
    Listen      string
    File        string
    Directory   string
    BufferLimit string
}

type Connection struct {
    File        string
    RemoteAddr  string
    ReadOnly    bool
}

var connections = make(map[string][]Connection)

/*
    Add a new connection to the list of connections for a file. Make sure there is only one writable connection per filename
    returns true if the connection was added correctly. false otherwise
 */
func addConnection(filename string, readOnly bool, remoteAddr string) bool {
    currentConnections, ok := connections[filename]
    if ok == false {
        currentConnections = make([]Connection, 4)
    }

    // If this a writable request, check to see if anybody else has a writable connection
    if !readOnly {
        for _, conn := range currentConnections {
            if !conn.ReadOnly {
                fmt.Printf("Error, too many writable connections. %s is already connected to %s\n", remoteAddr, filename)
                return false
            }
        }
    }

    newConnection := Connection{
        File: filename,
        RemoteAddr: remoteAddr,
        ReadOnly: readOnly,
    }

    connections[filename] = append(currentConnections, newConnection)
    return true
}



var globalSettings Settings = Settings {
    ReadOnly: false,
    AutoFlush: true,
    Host: "localhost",
    Port: "8000",
    Listen: "",
    File: "",
    Directory: "sample_disks",
    BufferLimit: "2048",
}

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

func export_name(output *bufio.Writer, conn net.Conn, payload_size int, payload []byte, options uint32, globalSettings Settings) {
    fmt.Printf("have request to bind to: %s\n", string(payload[:payload_size]))

    defer conn.Close()

    //todo add support for file specification

    var filename bytes.Buffer
    readOnly := false

    var current_directory = globalSettings.Directory
    var err error
    if current_directory == "" {
        current_directory, err = os.Getwd()
        utils.ErrorCheck(err, true)
    }
    filename.WriteString(current_directory)
    filename.WriteString(nbd_folder)
    filename.Write(payload[:payload_size])

    fmt.Printf("Opening file: %s\n", filename.String())

    fileMode := os.O_RDWR
    if globalSettings.ReadOnly || (options & utils.NBD_OPT_READ_ONLY != 0) {
        fmt.Printf("Read Only is set\n")
        fileMode = os.O_RDONLY
        readOnly = true
    }

    file, err := os.OpenFile(filename.String(), fileMode, 0644)

    utils.ErrorCheck(err, false)
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

    // if requested, pad with 124 zeros
    if (options & utils.NBD_FLAG_NO_ZEROES) != utils.NBD_FLAG_NO_ZEROES {
        offset += 124
    }

    _, err = output.Write(buffer[:offset])
    output.Flush()
    utils.ErrorCheck(err, false)
    if err != nil {
        return
    }

    buffer_limit, _ := strconv.Atoi(globalSettings.BufferLimit)
    buffer_limit *= 1024    // set the buffer to 2mb

    buffer = make([]byte, buffer_limit)
    conn_reader := bufio.NewReader(conn)
    for {
        waiting_for := 28       // wait for at least the minimum payload size

        _, err := io.ReadFull(conn_reader, buffer[:waiting_for])
        if err == io.EOF {
            fmt.Printf("Abort detected, escaping processing loop\n")
            break
        }
        utils.ErrorCheck(err, true)

        //magic := binary.BigEndian.Uint32(buffer)
        command := binary.BigEndian.Uint32(buffer[4:8])
        //handle := binary.BigEndian.Uint64(buffer[8:16])
        from := binary.BigEndian.Uint64(buffer[16:24])
        length := binary.BigEndian.Uint32(buffer[24:28])

        // Error out and drop the connection if there is an attempt to read too much
        if int(length) > buffer_limit {
            fmt.Printf("E")

            file.Sync()
            return
        }

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
            utils.ErrorCheck(err, true)

            binary.BigEndian.PutUint32(buffer[:4], utils.NBD_REPLY_MAGIC)
            binary.BigEndian.PutUint32(buffer[4:8], 0)                      // error bits

            conn.Write(buffer[:16+length])

            continue
        case utils.NBD_COMMAND_WRITE:
            if readOnly {
                fmt.Printf("E")
                fmt.Printf("\nAttempt to write to read only file blocked\n")

                continue
            }

            fmt.Printf("W")

            _, err := io.ReadFull(conn_reader, buffer[28:28+length])
            if err == io.EOF {
                fmt.Printf("Abort detected, escaping processing loop\n")
                break
            }
            utils.ErrorCheck(err, true)

            _, err = file.WriteAt(buffer[28:28+length], int64(from))
            utils.ErrorCheck(err, true)

            if globalSettings.AutoFlush {
                file.Sync()
            }

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

/*
First check for a specific file. If one is specified, use it. If not, check for a directory. If that is not
available, use the CWD.
 */
func send_export_list(output *bufio.Writer, options uint32, globalSettings Settings) {
    if globalSettings.File != "" {
        _, file := filepath.Split(globalSettings.File)

        send_export_list_item(output, options, file)
        send_ack(output, options)
        return
    }

    var current_directory string
    var err error
    if globalSettings.Directory == "" {
        current_directory, err = os.Getwd()
        utils.ErrorCheck(err, true)
    }

    files, err := ioutil.ReadDir(current_directory + nbd_folder)
    utils.ErrorCheck(err, true)

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
    app := cli.NewApp()
    app.Name = "AnyBlox"
    app.Usage = "block storage for the masses"
    app.Action = func(c *cli.Context) error {
        globalSettings.Host = c.GlobalString("host")
        globalSettings.Port = c.GlobalString("port")
        globalSettings.Listen = c.GlobalString("listen")
        globalSettings.File = c.GlobalString("file")
        globalSettings.Directory = c.GlobalString("directory")
        globalSettings.BufferLimit = c.GlobalString("buffer")

        if globalSettings.Listen == "" && (globalSettings.Host == "" || globalSettings.Port == "") {
            fmt.Println("Please specify either a full 'listen' parameter (e.g. 'localhost:8000', '192.168.1.2:8000) or a host and port\n")
        }
        return nil
    }

    app.Flags = []cli.Flag {
        cli.StringFlag{
            Name: "host",
            Value: globalSettings.Host,
            Usage: "Hostname or IP address you want to serve traffic on. e.x. 'localhost', '192.168.1.2'",
            //Destination: &globalSettings.Host,
        },
        cli.StringFlag{
            Name: "port",
            Value: globalSettings.Port,
            Usage: "Port you want to serve traffic on. e.x. '8000'",
            //Destination: &globalSettings.Port,
        },
        cli.StringFlag{
            Name: "listen, l",
            //Destination: &globalSettings.Listen,
            Usage: "Address and port the server should listen on. Listen will take priority over host and port parameters. hostname:port - e.x. 'localhost:8000', '192.168.1.2:8000'",
        },
        cli.StringFlag{
            Name: "file, f",
            //Destination: &globalSettings.File,
            Value: "",
            Usage: "The file that should be shared by this server. 'file' overrides 'directory'. It is required to be a full absolute path that includes the filename",
        },
        cli.StringFlag{
            Name: "directory, d",
            //Destination: &globalSettings.Directory,
            Value: globalSettings.Directory,
            Usage: "Specify a directory where the files to share are located. Default is 'sample_disks",
        },
        cli.StringFlag{
            Name: "buffer",
            Value: globalSettings.BufferLimit,
            Usage: "The number of kilobytes in size of the maximum supported read request e.x. '2048'",
            //Destination: &globalSettings.BufferLimit,
        },
    }

    app.Run(os.Args)

    // Determine where the host should be listening to, depending on the arguments
    fmt.Printf("Parameter Check: listen (%s) host (%s) port (%s)\n", globalSettings.Listen, globalSettings.Host, globalSettings.Port)
    hostingAddress := globalSettings.Listen
    if len(globalSettings.Listen) == 0 {
        if len(globalSettings.Host) == 0 || len(globalSettings.Port) == 0 {
            panic("You need to specify a host and port or specify a listen address (host:port)\n")
        }
        fmt.Printf("the port is: %s\n", globalSettings.Port)

        port := string(globalSettings.Port)
        //fmt.Sprintf(port, "%d", globalSettings.Port)
        fmt.Printf("the port is now: %d\n", port)

        hostingAddress = globalSettings.Host + ":" + port
        fmt.Printf("The hosting address is %s, port is %s\n", hostingAddress, port)
    }

    fmt.Printf("aBlox server online at: %s\n", hostingAddress)
    listener, err := net.Listen("tcp", hostingAddress)

    utils.ErrorCheck(err, true)


    reply_magic := make([]byte, 4)
    binary.BigEndian.PutUint32(reply_magic, utils.NBD_REPLY_MAGIC)

    defer fmt.Printf("End of line\n")

    for {
        conn, err := listener.Accept()
        utils.ErrorCheck(err, false)
        if err != nil {
            continue
        }

        fmt.Printf("We have a new connection from: %s\n", conn.RemoteAddr())
        output := bufio.NewWriter(conn)

        output.WriteString("NBDMAGIC")      // init password
        output.WriteString("IHAVEOPT")      // Magic

        output.Write(defaultOptions)

        output.Flush()

        // Fetch the data until we get the initial options
        data := make([]byte, 1024)
        offset := 0
        waiting_for := 16       // wait for at least the minimum payload size

        _, err = io.ReadFull(conn, data[:waiting_for])
        utils.ErrorCheck(err, false)
        if err != nil {
            continue
        }

        options := binary.BigEndian.Uint32(data[:4])
        command := binary.BigEndian.Uint32(data[12:16])

        // If we are requesting an export, make sure we have the length of the data for the export name.
        if binary.BigEndian.Uint32(data[12:]) == utils.NBD_COMMAND_EXPORT_NAME {
            waiting_for += 4
            _, err = io.ReadFull(conn, data[16:20])
            utils.ErrorCheck(err, false)
            if err != nil {
                continue
            }
        }
        payload_size := int(binary.BigEndian.Uint32(data[16:]))

        offset = waiting_for
        waiting_for += int(payload_size)
        _, err = io.ReadFull(conn, data[offset:waiting_for])
        utils.ErrorCheck(err, false)
        if err != nil {
            continue
        }

        payload := make([]byte, payload_size)
        if payload_size > 0 {
            copy(payload, data[20:])
        }

        utils.LogData("Payload is:", payload_size, payload)

        // At this point, we have the command, payload size, and payload.
        switch command {
        case utils.NBD_COMMAND_LIST:
            send_export_list(output, options, globalSettings)
            conn.Close()
            break
        case utils.NBD_COMMAND_EXPORT_NAME:
            go export_name(output, conn, payload_size, payload, options, globalSettings)
            break
        }
    }

}
