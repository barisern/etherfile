package etherfile

import (
	"encoding/binary"
	"io/ioutil"
	"log"
	"net"
	"os"

	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
)

const partSize = 4
const maxPartBytes = (1<<(8*partSize) - 1)
const chunkSize = 1496
const myEtherType = 0xB33F

type Payload struct {
	partNo   []byte
	filename []byte // first byte is len of filename
	data     []byte
}

var fileParts []Payload

func splitFile(filename string) []Payload {
	var payloadArray []Payload

	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil
	}

	var partNo uint32 = 0
	for start := 0; start < len(bytes); start += chunkSize {
		end := start + chunkSize

		if end > len(bytes) {
			end = len(bytes)
		}

		partNoBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(partNoBytes, partNo)

		if partNo == 0 {
			filenameArray := []byte(filename)
			filenameArray = append([]byte{byte(len(filename))}, filenameArray...)

			end -= len(filename) + 1
			payloadArray = append(payloadArray, Payload{
				partNo:   partNoBytes,
				filename: filenameArray,
				data:     bytes[start:end],
			})
		} else {
			payloadArray = append(payloadArray, Payload{
				partNo:   partNoBytes,
				filename: nil,
				data:     bytes[start:end],
			})
		}

		if partNo != maxPartBytes {
			partNo++
		}

		//log.Println(payloadArray[partNo])
	}

	return payloadArray
}

func SendPacket() {
	fileParts = splitFile("5gb.test")

	for i, part := range fileParts {
		payload := append(part.partNo, part.filename...)
		payload = append(payload, part.data...)

		f := &ethernet.Frame{
			Destination: net.HardwareAddr{0x00, 0x0c, 0x29, 0x88, 0x59, 0xd9}, // kali 2 - 00:0c:29:88:59:d9
			Source:      net.HardwareAddr{0x00, 0x0c, 0x29, 0x14, 0x60, 0x71}, // kali 1 - 00:0c:29:14:60:71
			EtherType:   myEtherType,
			Payload:     payload,
		}

		b, err := f.MarshalBinary()
		if err != nil {
			log.Fatalf("failed to marshal frame: %v", err)
		}
		sendEthernetFrame(b)

		log.Printf("Sent packet: %d\n", i)
	}
}

func sendEthernetFrame(packet []byte) {
	iface, err := net.InterfaceByName("eth0")
	if err != nil {
		log.Fatalf("failed to open interface: %v", err)
	}

	conn, err := raw.ListenPacket(iface, myEtherType, nil)
	if err != nil {
		log.Fatalf("failed to listen %v", err)
	}
	defer conn.Close()
	// kali 2 - 00:0c:29:88:59:d9
	addr := &raw.Addr{HardwareAddr: net.HardwareAddr{0x00, 0x0c, 0x29, 0x88, 0x59, 0xd9}}
	if _, err := conn.WriteTo(packet, addr); err != nil {
		log.Fatalf("failed to write frame: %v", err)
	}
}

func ListenPacket() {
	iface, err := net.InterfaceByName("eth0")
	if err != nil {
		log.Fatalf("failed to open interface: %v", err)
	}

	conn, err := raw.ListenPacket(iface, myEtherType, nil)
	if err != nil {
		log.Fatalf("failed to listen %v", err)
	}
	defer conn.Close()

	bytes := make([]byte, iface.MTU)
	var f ethernet.Frame
	var packet int = 0

	for {
		log.Println("Listening!")

		n, _, err := conn.ReadFrom(bytes)
		if err != nil {
			log.Fatalf("failed to receive message: %v", err)
		}

		if err := (&f).UnmarshalBinary(bytes[:n]); err != nil {
			log.Fatalf("failed to unmarshal ethernet frame: %v", err)
		}

		if packet == 0 {
			filenameLen := int(f.Payload[partSize : partSize+1][0]) + 1
			saveFile(Payload{f.Payload[:partSize], f.Payload[partSize : partSize+filenameLen], f.Payload[partSize+filenameLen:]})
			packet++
		} else {
			saveFile(Payload{f.Payload[:partSize], nil, f.Payload[partSize:]})
			packet++
		}

		// log.Printf("[%s] %s", addr.String(), string(f.Payload))
	}
}

var fname string

func saveFile(payload Payload) {
	var f *os.File
	partNo := binary.BigEndian.Uint32(payload.partNo)

	if partNo > 0 {
		f, _ = os.OpenFile(fname, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if _, err := f.Write(payload.data); err != nil {
			log.Fatal(err)
		}
	} else {
		fname = string(payload.filename[1:])
		f, _ = os.OpenFile(fname, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)

		if _, err := f.Write(payload.data); err != nil {
			log.Fatal(err)
		}
	}
	f.Close()
}
