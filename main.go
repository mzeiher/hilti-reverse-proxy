package main

import (
	"context"
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
				srcConnection.Close()
				return
			}
			defer remoteConnection.Close()
			defer srcConnection.Close()

			forwarder := NewForwarder(*srcConnection, *remoteConnection)

			fmt.Fprintf(os.Stdout, "Started forwarding from %s to %s\n", srcConnection.LocalAddr().String(), remoteConnection.RemoteAddr().String())

			err = forwarder.Start()

			fmt.Fprintf(os.Stdout, "Stopped forwarding from %s to %s, error: %s\n", srcConnection.LocalAddr().String(), remoteConnection.RemoteAddr().String(), err)
		}()
	}
}

type Forwarder struct {
	src       net.TCPConn
	dst       net.TCPConn
	lastError error
}

func NewForwarder(src net.TCPConn, dst net.TCPConn) *Forwarder {
	return &Forwarder{
		src:       src,
		dst:       dst,
		lastError: nil,
	}
}

func (fw *Forwarder) Start() error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	go fw.Forward(&fw.src, &fw.dst, ctx, cancelFunc)
	go fw.Forward(&fw.dst, &fw.src, ctx, cancelFunc)
	<-ctx.Done()
	return fw.lastError
}

func (fw *Forwarder) Forward(in io.Reader, out io.Writer, ctx context.Context, cancel context.CancelFunc) {
	buffer := make([]byte, 1024)
	for {
		read, err := in.Read(buffer)
		if err != nil {
			fw.lastError = err
			cancel()
			return
		}
		_, err = out.Write(buffer[:read])
		if err != nil {
			fw.lastError = err
			cancel()
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}
