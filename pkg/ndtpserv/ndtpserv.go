package ndtpserv

import (
	"github.com/ashirko/navprot/pkg/ndtp"
	"log"
	"net"
	"sync"
	"time"
)

type Result struct {
	numConn    uint64
	numReceive int
	numControl int
}

var (
	running       bool
	muRun         sync.Mutex
	newConnChan   chan uint64
	closeConn     chan Result
	controlPacket = []byte{126, 126, 12, 0, 2, 0, 37, 196, 2, 0, 0, 0, 0, 0, 0, 0, 0, 110, 0, 1, 0, 0, 0, 0, 0, 6, 0}
)

const (
	defaultBufferSize = 1024
	writeTimeout      = 10 * time.Second
	readTimeout       = 180 * time.Second
)

func Start(listenPort string, mode int, num int) {
	listenAddress := "localhost:" + listenPort
	l, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Printf("error while listening: %s", err)
		return
	}
	running = true
	defer l.Close()
	log.Printf("NDTP server was started. Listen address: %v; Mode: %d; Number packets to receive: %v",
		listenAddress, mode, num)
	newConnChan = make(chan uint64)
	closeConn = make(chan Result)
	go func() {
		connNo := uint64(1)
		for {
			log.Printf("wait accept...")
			c, err := l.Accept()
			if err != nil {
				muRun.Lock()
				if !running {
					muRun.Unlock()
					return
				}
				muRun.Unlock()
				log.Printf("error while accepting: %s", err)
				return
			}
			defer c.Close()
			log.Printf("accepted connection %d (%s <-> %s)", connNo, c.RemoteAddr(), c.LocalAddr())
			go handleConnection(c, connNo, mode, num)
			newConnChan <- connNo
			connNo++
		}
	}()
	results := waitStop()
	muRun.Lock()
	running = false
	muRun.Unlock()
	log.Printf("NDTP server has completed work")
	for _, r := range results {
		endingReceive := ""
		isControl := ""
		if r.numReceive > 1 {
			endingReceive = "s"
		}
		if r.numControl == 1 {
			isControl = ", sent 1 control packet"
		}
		log.Printf("For connection %d: received %d data packet"+endingReceive+isControl,
			r.numConn, r.numReceive)
	}
}

func waitStop() []Result {
	numsConn := uint64(0)
	results := make([]Result, 0)
	for {
		select {
		case <-newConnChan:
			numsConn++
		case res := <-closeConn:
			results = append(results, res)
			numsConn--
			if numsConn == 0 {
				return results
			}
		}
	}
}

func handleConnection(conn net.Conn, connNo uint64, mode int, num int) {
	res := Result{connNo, 0, 0}
	err := waitFirstMessage(conn, res)
	if err != nil {
		closeConn <- res
		return
	}
	receiveData(conn, connNo, num, &res)
	if mode == 1 {
    		err = sendControlPacket(conn, &res)
    		if err != nil {
    			closeConn <- res
    			return
    		}
    	}
	closeConn <- res
}

func waitFirstMessage(conn net.Conn, res Result) (err error) {
	err = conn.SetReadDeadline(time.Now().Add(readTimeout))
	if err != nil {
		log.Printf("can't set read deadline %s", err)
	}
	var b [defaultBufferSize]byte
	n, err := conn.Read(b[:])
	if err != nil {
		log.Printf("can't get first message from client: %s", err)
		return
	}
	log.Printf("got first message %v from client %s", b[:n], conn.RemoteAddr())
	parsedPacket := new(ndtp.Packet)
	_, err = parsedPacket.Parse(b[:n])
	if err != nil {
		log.Printf("can't parse first message from client: %s", err)
		return
	}
	log.Printf("parsed first message: %v", parsedPacket.String())
	reply := parsedPacket.Reply(ndtp.NphResultOk)
	err = send(conn, reply)
	if err != nil {
		log.Printf("can't send reply for first message: %s", err)
		return
	}
	log.Printf("send reply for first message: %v", parsedPacket.String())
	return
}

func sendControlPacket(conn net.Conn, res *Result) (err error) {
	parsedPacket := new(ndtp.Packet)
	_, err = parsedPacket.Parse(controlPacket)
	if err != nil {
		log.Printf("error parse ndtp control packet: %v", err)
		return
	}
	log.Printf("send ndtp control packet: %s", parsedPacket.String())
	err = send(conn, controlPacket)
	if err != nil {
		log.Printf("error send ndtp control packet: %v", err)
		return
	}
	res.numControl++
	return
}

func receiveData(conn net.Conn, connNo uint64, numPacketsToReceive int, res *Result) {
	var restBuf []byte
	for res.numReceive < numPacketsToReceive {
		err := conn.SetReadDeadline(time.Now().Add(readTimeout))
		if err != nil {
			log.Printf("can't set read deadline %s", err)
		}
		var buf [defaultBufferSize]byte
		n, err := conn.Read(buf[:])
		if err != nil {
			log.Printf("can't get data from connection %d: %v", connNo, err)
			break
		}
		log.Printf("read n byte: %d",n)
		restBuf = append(restBuf, buf[:n]...)
		for len(restBuf) != 0 {
			parsedPacket := new(ndtp.Packet)
			restBuf, err = parsedPacket.Parse(restBuf)
			if err != nil {
				log.Printf("error while parsing NDTP: %v", err)
				break
			}
			log.Printf("receive ndtp packet: %s", parsedPacket.String())
			res.numReceive++
			reply := parsedPacket.Reply(ndtp.NphResultOk)
			err = send(conn, reply)
			if err != nil {
				log.Printf(" error while send response to %d: %v", connNo, err)
				break
			}
			log.Printf("send reply: %s", parsedPacket.String())
		}
	}
	log.Printf("finish receive numReceive: %d",res.numReceive)
}

func send(conn net.Conn, packet []byte) error {
	err := conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err != nil {
		return err
	}
	_, err = conn.Write(packet)
	return err
}
