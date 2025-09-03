// TODO: Add package documentation for `main`, like this:
// Package main something something...
package main

import (
	// "d7024e/kademlia"
	"fmt"
	"net"
	"log"
)

func main() {
	fmt.Println("Pretending to run the kademlia app...")
	// Using stuff from the kademlia package here. Something like...
	// id := kademlia.NewKademliaID("FFFFFFFF00000000000000000000000000000000")
	// contact := kademlia.NewContact(id, "localhost:8000")
	// fmt.Println(contact.String())
	// fmt.Printf("%v\n", contact)

	conn, err := net.Dial("udp", "8.8.8.8:80")
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    localAddr := conn.LocalAddr().(*net.UDPAddr)

    fmt.Printf("%v\n", localAddr.IP)


	for {

	}

	return

	// port_s := stronv.FormatInt(25565)

	// ln, err := net.Listen("udp", ":" + port_s)
	// if (err != nil) {
	// 	panic("failed to listen")
	// }

	// for {
	// 	buf := make([]byte, 10)
	// 	conn, err := ln.Accept()
	// 	if err != nil {
	// 		panic("failed accept")
	// 	}

	// 	conn.Read(buf)
	// 	fmt.printf("%v", buf)

	// 	conn.Close()
	// }
}
