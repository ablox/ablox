// Copyright 2016 Jacob Taylor jacob.taylor@gmail.com
// License: Apache2

package main

import (
	"fmt"
	"log"
	"net/http"
	"net"
	"os"
	"bufio"
    "encoding/binary"
)

const (
    NBD_REQUEST_MAGIC = 0x25609513
    NBD_REPLY_MAGIC = 0x67446698
)

type NBDRequest struct {
    magic       uint32
    reqtype     uint32
    handle      uint64
    from        uint64
    length      uint32
}

func createRequest() NBDRequest {
    var request NBDRequest
    request.magic = NBD_REQUEST_MAGIC
    request.reqtype = 0
    request.from = 0
    request.handle = 0
    request.length = 0

    return request
}

func (r NBDRequest) encodeRequest(data []byte){
	binary.BigEndian.PutUint32(data, r.magic)
	binary.BigEndian.PutUint32(data[4:8], r.reqtype)
	binary.BigEndian.PutUint64(data[9:17], r.handle)
	binary.BigEndian.PutUint64(data[18:26], r.from)
	binary.BigEndian.PutUint32(data[27:32], r.length)
}

func main() {
	tcpAddr, err := net.ResolveTCPAddr("tcp", "jacobu.local:8000")
	errorCheck(err)
	conn, err := net.Dial("tcp", tcpAddr.String())
	errorCheck(err)

	fmt.Println("We are connectd to: %s", tcpAddr.String())
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

    data := make([]byte, 1000)
    request := createRequest()

    count, err := reader.Read(data)
    errorCheck(err)
    logData("A", count, data)

    data = make([]byte, 1000)
    request.reqtype = 3
    request.encodeRequest(data)

    newline := make([]byte, 1)
    newline[0] = byte('\n')
    writer.Write(newline)
    writer.Write(data[0:32])
    writer.Flush()

    count, err = reader.Read(data)
    errorCheck(err)
    logData("B", count, data)

    os.Exit(0)

	fmt.Println("Starting Now!")
	http.HandleFunc("/", receive)
	http.HandleFunc("/start", start)
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

func logData(msg string, count int, data []byte) {
    fmt.Printf("%5s (count %3d) Data: '%s' (%v)\n", msg, count, string(data[0:count]), data[0:count])
}

func errorCheck(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encountered: %v", err)
		os.Exit(0)
	}
}

func receive(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "<html>\n")
	fmt.Fprintf(w, "Hello: %q\n", r.URL.Path)
	fmt.Fprintf(w, "</html>\n")
}

func start(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<html>starting a new process<HR><a href='/'>back</a></html>")
}
