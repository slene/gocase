package main

import (
    "bufio"
    "encoding/binary"
    "flag"
    "io"
    "log"
    "net"
    "os"
    "time"
)

func send(conn *net.TCPConn, file string) {
    fh, err := os.Open(file)
    if err != nil {
        log.Fatalln(err)
    }
    stat, _ := fh.Stat()
    b := make([]byte, 8)
    binary.BigEndian.PutUint64(b, uint64(stat.Size()))
    _, err = conn.Write(b)
    if err != nil {
        log.Fatal(err)
    }
    r := bufio.NewReader(fh)
    n, err := conn.ReadFrom(r)
    if err != nil {
        log.Fatalln("send err ", err)
    }
    log.Println("file send size: ", n)
}

func reveive(conn *net.TCPConn, file string) {
    b := make([]byte, 8)
    n, err := conn.Read(b)
    if err != nil {
        log.Fatalln(err)
    }
    if n != 8 {
        log.Fatalln("err header size ", n)
    }
    size := binary.BigEndian.Uint64(b)
    src := bufio.NewReader(conn)
    fh, err := os.Create(file)
    dst := bufio.NewWriter(fh)
    io.CopyN(dst, src, int64(size))
    dst.Flush()
    fh.Close()
}

// ./copyfile -mode=server -out=/outFile
// ./copyfile -mode=client -out=/inFile
func main() {
    var mode string
    var filein string
    var fileout string
    flag.StringVar(&mode, "mode", "server", "server or client")
    flag.StringVar(&filein, "in", "", "full file path")
    flag.StringVar(&fileout, "out", "", "full file path")
    flag.Parse()

    saddr, _ := net.ResolveTCPAddr("tcp4", ":8080")

    if mode == "server" {

        lis, err := net.ListenTCP("tcp4", saddr)
        if err != nil {
            log.Fatalln(err)
        }
        for {
            log.Println("wait connect")
            conn, err := lis.Accept()
            if err != nil {
                log.Println(err)
                time.Sleep(1 * time.Millisecond)
                continue
            }
            log.Println("connected ", conn.RemoteAddr())
            reveive(conn.(*net.TCPConn), fileout)
        }

    } else {

        conn, err := net.DialTCP("tcp4", nil, saddr)
        if err != nil {
            log.Fatalln(err)
        }

        send(conn, filein)
    }
}
