package ndtpserv

import (
	"log"
	"net"
	"time"

	"github.com/ashirko/navprot/pkg/ndtp"
)

type Stat struct {
	numData    int
	numOldData int
	numReply   int
}

const (
	defaultBufferSize = 1024
	writeTimeout      = 10 * time.Second
	readTimeout       = 180 * time.Second
)

var chanStat chan Stat

func Start(listenAddress string) {
	l, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Printf("error while listening: %s", err)
		return
	}
	defer l.Close()
	chanStat = make(chan Stat)
	go waitResult()

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
	//	log.Printf("got first message %v from client %s", b[:n], conn.RemoteAddr())
	parsedPacket := new(ndtp.Packet)
	_, err = parsedPacket.Parse(b[:n])
	if err != nil {
		log.Printf("can't parse first message from client: %s", err)
		return
	}
	//log.Printf("parsed first message: %v", parsedPacket.String())
	reply := parsedPacket.Reply(ndtp.NphResultOk)
	err = send(conn, reply)
	if err != nil {
		log.Printf("can't send reply for first message: %s", err)
		return
	}
	//log.Printf("send reply for first message: %v", parsedPacket.String())
	return
}

func receiveData(conn net.Conn, connNo uint64) {
	var stat Stat
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	var restBuf []byte
	for {
		select {
		case <-ticker.C:
			chanStat <- stat
			stat.numData = 0
			stat.numOldData = 0
			stat.numReply = 0
		default:
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
			//	log.Printf("read n byte: %d", n)
			restBuf = append(restBuf, buf[:n]...)
			for len(restBuf) != 0 {
				parsedPacket := new(ndtp.Packet)
				restBuf, err = parsedPacket.Parse(restBuf)
				if err != nil {
					log.Printf("error while parsing NDTP: %v", err)
					break
				}
				if parsedPacket.PacketType() == ndtp.NphSndHistory {
					stat.numOldData++
				} else {
					stat.numData++
				}
				//	log.Printf("receive ndtp packet: %s", parsedPacket.String())
				reply := parsedPacket.Reply(ndtp.NphResultOk)
				err = send(conn, reply)
				if err != nil {
					log.Printf(" error while send response to %d: %v", connNo, err)
					break
				}
				stat.numReply++
				//	log.Printf("send reply: %s", parsedPacket.String())
			}

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

func waitResult() {
	var totalStat Stat
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case stat := <-chanStat:
			totalStat.numData = totalStat.numData + stat.numData
			totalStat.numOldData = totalStat.numOldData + stat.numOldData
			totalStat.numReply = totalStat.numReply + stat.numReply
		case <-ticker.C:
			log.Printf("last minute: receive data %v (realtime %v, old %v), send reply %v",
				totalStat.numData+totalStat.numOldData, totalStat.numData, totalStat.numOldData, totalStat.numReply)
			totalStat.numData = 0
			totalStat.numOldData = 0
			totalStat.numReply = 0
		}
	}
}
