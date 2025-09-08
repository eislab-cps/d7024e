package kademlia

import (
	"net"
	"log"
	"fmt"
	"encoding/binary"
)

type RPCType uint8

const (
	RPCTypeInvalid = iota
	RPCTypePingReply
	RPCTypePing
	RPCTypeStore
	RPCTypeFindNode
	RPCTypeFindValue
)

type RPCError uint8
const (
	RPCErrorNoError = iota
	RPCErrorLackOfSpace
)

type Network struct {
}

type RPC struct {
	typ RPCType
	id KademliaID
	error RPCError
	data_size uint64
	data []byte
}

func Listen(ip string, port int) {

	addr := net.UDPAddr{Port: port, IP: net.ParseIP(ip)}
	
	for {
		fmt.Printf("listening...\n")
		conn, err := net.ListenUDP("udp", &addr)
		if (err != nil) {
			log.Fatalf("Failed to listen %v\n", err)
		}
		buf := make([]byte, 1000)
		
		n, rec_addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Fatalf("Failed to read packet %v\n", err)
		}

		s_buf := string(buf[0:n - 1])

		fmt.Printf("Received %v bytes %v\n", n, s_buf)
		
		if (s_buf == "ping") {
			fmt.Printf("writing ping...\n")
			_, err := conn.WriteTo([]byte("pong"), rec_addr)
			if err != nil {
				log.Fatalf("write error %v\n", err)
			}
			continue
		}
		
		var rpc RPC
		binary.Decode(buf, binary.BigEndian, rpc)

		// receiving
		switch rpc.typ {
			case RPCTypePingReply: {
				log.Printf("ping reply\n")
				// update bucket
				panic("TODO update bucket when receiving ping reply")
			}
			case RPCTypePing: {
				var rpc RPC
				rpc.typ = RPCTypePingReply
				rpc.id = *NewRandomKademliaID()
				write_buf, err := binary.Append(nil, binary.BigEndian, rpc)
				if err != nil {
					log.Fatalf("Failed %v\n", err)
				}
				_, err = conn.WriteTo(write_buf, rec_addr)
				if err != nil {
					log.Fatalf("RPCPing write error %v\n", err)
				}
			}
		}
		conn.Close()
	}
}

func (network *Network) SendPingMessage(contact *Contact) {
	var rpc RPC
	rpc.typ = RPCTypePing
	rpc.id = *NewRandomKademliaID()

	write_buf, err := binary.Append(nil, binary.BigEndian , rpc)


	addr := net.UDPAddr{Port: 8000, IP: net.ParseIP(contact.Address)}
	conn, err := net.DialUDP("udp", nil, &addr)
	if err != nil {
		log.Fatalf("Failed to send ping message, %v\n", err)
	}
	defer conn.Close()
	_, err = conn.Write(write_buf)
	if err != nil {
		log.Fatalf("write error %v\n", err)
	}
}

func (network *Network) SendFindContactMessage(contact *Contact) {
	panic("TODO")
}

func (network *Network) SendFindDataMessage(hash string) {
	panic("TODO")
}

func (network *Network) SendStoreMessage(key KademliaID, data []byte) {
	var rpc RPC
	rpc.typ = RPCTypeStore
	rpc.id = *NewRandomKademliaID()

	rpc.data_size = uint64(len(data))
	rpc.data = data

	_, _ = binary.Append(nil, binary.BigEndian , rpc)


	panic("TODO add node lookup to retrieve closest node to key")
	// addr := net.UDPAddr{Port: 8000, IP: net.ParseIP(contact.Address)}
	// conn, err := net.DialUDP("udp", nil, &addr)
	// if err != nil {
	// 	log.Fatalf("Failed to send ping message, %v\n", err)
	// }
	// defer conn.Close()
	// _, err = conn.Write(write_buf)
	// if err != nil {
	// 	log.Fatalf("write error %v\n", err)
	// }
}
