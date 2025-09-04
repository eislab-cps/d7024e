// TODO: Add package documentation for `main`, like this:
// Package main something something...
package main

import (
	// "d7024e/kademlia"
	"fmt"
	"net"
	"log"
)


func server(ip string, port int) {
	addr := net.UDPAddr{Port: port, IP: net.ParseIP(ip)}
	conn, err := net.ListenUDP("udp", &addr)
	if (err != nil) {
		log.Fatalf("Failed to listen %v\n", err)
	}
	defer conn.Close()
	for {
		buf := make([]byte, 100)
		
		n, err := conn.Read(buf)
		if err != nil {
			log.Fatalf("Failed to read packet %v\n", err)
		}
		fmt.Printf("Received %v bytes %v\n", n, string(buf))
	}
}

func main() {
	fmt.Println("Pretending to run the kademlia app...")
	// Using stuff from the kademlia package here. Something like...
	// id := kademlia.NewKademliaID("FFFFFFFF00000000000000000000000000000000")
	// contact := kademlia.NewContact(id, "localhost:8000")
	// fmt.Println(contact.String())
	// fmt.Printf("%v\n", contact)


	server("0.0.0.0", 8000)
	fmt.Printf("left loop\n")
	for {}
}
