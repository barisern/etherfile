package main

import (
	"fmt"

	etherfile "github.com/barisern/etherfile/lib"
)

func main() {
	var selection string

	fmt.Print("Send or listen? (s/l): ")
	fmt.Scanf("%s", &selection)

	if selection == "s" {
		etherfile.SendPacket()
	} else {
		etherfile.ListenPacket()
	}
}
