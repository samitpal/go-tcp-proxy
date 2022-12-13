package proxy

import (
	"io"
	"net"
)

// Proxy - Manages a Proxy connection, piping data between local and remote.
type Proxy struct {
	sentBytes             uint64
	receivedBytes         uint64
	laddr, r1addr, r2addr *net.TCPAddr
	lconn, r1conn, r2conn io.ReadWriteCloser
	erred                 bool
	errsig                chan bool
	// this is to hold a copy of the local socket data to be sent to second remote
	buffSecondRemote chan []byte
	// Settings
	Nagles    bool
	Log       Logger
	OutputHex bool
}

// New - Create a new Proxy instance. Takes over local connection passed in,
// and closes it when finished.
func New(lconn *net.TCPConn, laddr, r1addr *net.TCPAddr, r2addr *net.TCPAddr) *Proxy {
	return &Proxy{
		lconn:            lconn,
		laddr:            laddr,
		r1addr:           r1addr,
		r2addr:           r2addr,
		erred:            false,
		errsig:           make(chan bool),
		buffSecondRemote: make(chan []byte, 0xffff),
		Log:              NullLogger{},
	}
}

type setNoDelayer interface {
	SetNoDelay(bool) error
}

// Start - open connection to remote and start proxying data.
func (p *Proxy) Start() {
	defer p.lconn.Close()

	var err error
	//connect to first remote
	p.r1conn, err = net.DialTCP("tcp", nil, p.r1addr)
	if err != nil {
		p.Log.Warn("Remote connection failed: %s", err)
		return
	}
	defer p.r1conn.Close()

	//connect to second remote
	p.r2conn, err = net.DialTCP("tcp", nil, p.r2addr)
	if err != nil {
		p.Log.Warn("Second Remote connection failed: %s", err)
		return
	}
	defer p.r2conn.Close()
	
	//nagles?
	if p.Nagles {
		if conn, ok := p.lconn.(setNoDelayer); ok {
			conn.SetNoDelay(true)
		}
		if conn, ok := p.r1conn.(setNoDelayer); ok {
			conn.SetNoDelay(true)
		}
		if conn, ok := p.r2conn.(setNoDelayer); ok {
			conn.SetNoDelay(true)
		}
	}

	//display both ends
	p.Log.Info("Opened %s >>> %s", p.laddr.String(), p.r1addr.String())

	//bidirectional copy to first remote
	go p.pipe(p.lconn, p.r1conn)
	go p.pipe(p.r1conn, p.lconn)

	// close the buffered channel meant for second remote
	defer close(p.buffSecondRemote)
	// one way copy to second remote
	go p.pipeToSecondRemote(p.r2conn)

	//wait for close...
	<-p.errsig
	p.Log.Info("Closed (%d bytes sent, %d bytes recieved)", p.sentBytes, p.receivedBytes)
}

func (p *Proxy) err(s string, err error) {
	if p.erred {
		return
	}
	if err != io.EOF {
		p.Log.Warn(s, err)
	}
	p.errsig <- true
	p.erred = true
}

func (p *Proxy) pipe(src, dst io.ReadWriter) {
	islocal := src == p.lconn

	var dataDirection string
	if islocal {
		dataDirection = ">>> %d bytes sent%s"
	} else {
		dataDirection = "<<< %d bytes recieved%s"
	}

	var byteFormat string
	if p.OutputHex {
		byteFormat = "%x"
	} else {
		byteFormat = "%s"
	}

	//directional copy (64k buffer)
	buff := make([]byte, 0xffff)
	for {
		n, err := src.Read(buff)
		if err != nil {
			p.err("Read failed '%s'\n", err)
			return
		}
		b := buff[:n]

		//show output
		p.Log.Debug(dataDirection, n, "")
		p.Log.Trace(byteFormat, b)

		// send a copy of the socket data to the channel meant for the second remote
		go func() {
			if islocal {
				buff1 := buff[:n]
				p.buffSecondRemote <- buff1
			}
		}()
		//write out result
		n, err = dst.Write(b)
		if err != nil {
			p.err("Write failed '%s'\n", err)
			return
		}
		if islocal {
			p.sentBytes += uint64(n)
		} else {
			p.receivedBytes += uint64(n)
		}
	}
}

func (p *Proxy) pipeToSecondRemote(dst io.ReadWriter) {
	for d := range p.buffSecondRemote {
		_, err := dst.Write(d)
		if err != nil {
			p.err("Write failed to second remote '%s'\n", err)
			return
		}
	}
}
