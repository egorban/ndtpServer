package ndtpserv

import (
	"github.com/ashirko/navprot/pkg/ndtp"
	"log"
	"net"
	"time"
)

const (
	defaultBufferSize = 1024
	writeTimeout      = 10 * time.Second
	readTimeout       = 180 * time.Second
)

func Start(listenPort string) {
	listenAddress := "localhost:" + listenPort
	l, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Printf("error while listening: %s", err)
		return
	}
	defer l.Close()
	log.Printf("NDTP server was started. Listen address: %v", listenAddress)
	connNo := uint64(1)
	for {
		log.Printf("wait accept...")
		c, err := l.Accept()
		if err != nil {
			log.Printf("error while accepting: %s", err)
			return
		}
		defer c.Close()
		log.Printf("accepted connection %d (%s <-> %s)", connNo, c.RemoteAddr(), c.LocalAddr())
		go handleConnection(c, connNo)
		connNo++
	}
}

func handleConnection(conn net.Conn, connNo uint64) {
	err := waitFirstMessage(conn)
	if err != nil {
		return
	}
	receiveData(conn, connNo)
}

func waitFirstMessage(conn net.Conn) (err error) {
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

func receiveData(conn net.Conn, connNo uint64) {
	var restBuf []byte
	for {
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
		log.Printf("read n byte: %d", n)
		restBuf = append(restBuf, buf[:n]...)
		for len(restBuf) != 0 {
			parsedPacket := new(ndtp.Packet)
			restBuf, err = parsedPacket.Parse(restBuf)
			if err != nil {
				log.Printf("error while parsing NDTP: %v", err)
				break
			}
			log.Printf("receive ndtp packet: %s", parsedPacket.String())
			reply := parsedPacket.Reply(ndtp.NphResultOk)
			err = send(conn, reply)
			if err != nil {
				log.Printf(" error while send response to %d: %v", connNo, err)
				break
			}
			log.Printf("send reply: %s", parsedPacket.String())
		}
	}
}

func send(conn net.Conn, packet []byte) error {
	err := conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err != nil {
		return err
	}
	_, err = conn.Write(packet)
	return err
}
