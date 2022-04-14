package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	var srcPort int
	var srcHost string
	var dstPort int
	var dstHost string
	flag.IntVar(&srcPort, "port", 0, "Source Port")
	flag.StringVar(&srcHost, "network", "0.0.0.0", "Network to listen to")
	flag.IntVar(&dstPort, "dstPort", 0, "Destination port")
	flag.StringVar(&dstHost, "dstHost", "", "Desitnation ip")

	flag.Parse()

	if dstPort == 0 {
		fmt.Fprintf(os.Stderr, "Missing destination port\n")
		flag.Usage()
		os.Exit(-1)
	}
	if dstHost == "" {
		fmt.Fprintf(os.Stderr, "Missing destination host\n")
		flag.Usage()
		os.Exit(-1)
	}

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP(srcHost), Port: srcPort})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(-2)
	}
	fmt.Fprintf(os.Stdout, "Started listening on %s\n", listener.Addr().String())
	for {
		srcConnection, err := listener.AcceptTCP()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listening to address: %s\n", err)
			os.Exit(-2)
		}
		go func() {
			remoteConnection, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP(dstHost), Port: dstPort})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error establishing remote connection: %s\n", err)
			}
			forwarder := NewForwarder(*srcConnection, *remoteConnection)
			defer remoteConnection.Close()
			defer srcConnection.Close()

			fmt.Fprintf(os.Stdout, "Started forwarding from %s to %s\n", srcConnection.LocalAddr().String(), remoteConnection.RemoteAddr().String())

			err = forwarder.Start()

			fmt.Fprintf(os.Stdout, "Stopped forwarding from %s to %s, error: %s\n", srcConnection.LocalAddr().String(), remoteConnection.RemoteAddr().String(), err)
		}()
	}
}

type Forwarder struct {
	src        net.TCPConn
	dst        net.TCPConn
	errChannel chan error
	stop       bool
}

func NewForwarder(src net.TCPConn, dst net.TCPConn) *Forwarder {
	return &Forwarder{
		src:        src,
		dst:        dst,
		stop:       false,
		errChannel: make(chan error),
	}
}

func (fw *Forwarder) Start() error {
	go fw.Forward(&fw.src, &fw.dst)
	go fw.Forward(&fw.dst, &fw.src)
	err := <-fw.errChannel
	fw.stop = true
	return err
}

func (fw *Forwarder) Forward(in io.Reader, out io.Writer) {
	buffer := make([]byte, 1024)
	for {
		read, err := in.Read(buffer)
		if err != nil {
			fw.errChannel <- err
			break
		}
		_, err = out.Write(buffer[:read])
		if err != nil {
			fw.errChannel <- err
			break
		}
		if fw.stop {
			break
		}
	}
}
