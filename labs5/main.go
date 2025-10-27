package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"
)

func main() {
	iface := flag.String("i", "", "interface to capture on (required)")
	snaplen := flag.Int("snap", 65535, "snapshot length (bytes)")
	promisc := flag.Bool("promisc", true, "promiscuous mode")
	outfile := flag.String("w", "", "optional pcap output file (path)")
	count := flag.Int("c", 0, "stop after N packets (0 = forever)")
	hexdump := flag.Bool("hex", true, "print hex dump of packet payload")
	maxdump := flag.Int("max", 128, "max bytes to show in hex dump per packet")
	flag.Parse()

	if *iface == "" {
		fmt.Fprintln(os.Stderr, "You must provide an interface with -i. Use `ifconfig` to list interfaces on macOS.")
		flag.Usage()
		os.Exit(2)
	}

	handle, err := pcap.OpenLive(*iface, int32(*snaplen), *promisc, pcap.BlockForever)
	if err != nil {
		log.Fatalf("pcap open: %v", err)
	}
	defer handle.Close()

	var pcapWriter *pcapgo.Writer
	var pcapFile *os.File
	if *outfile != "" {
		pcapFile, err = os.Create(*outfile)
		if err != nil {
			log.Fatalf("create pcap file: %v", err)
		}
		defer pcapFile.Close()
		pcapWriter = pcapgo.NewWriter(pcapFile)
		pcapWriter.WriteFileHeader(uint32(*snaplen), handle.LinkType())
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packets := packetSource.Packets()

	printed := 0
	for packet := range packets {
		if packet == nil {
			continue
		}

		data := packet.Data()
		ts := time.Now().Format(time.RFC3339Nano)
		fmt.Printf("%s  len=%d\n", ts, len(data))

		if *hexdump {
			dumpLen := *maxdump
			if dumpLen > len(data) {
				dumpLen = len(data)
			}
			hexstr := hex.EncodeToString(data[:dumpLen])
			for i := 0; i < len(hexstr); i += 32 {
				end := i + 32
				if end > len(hexstr) {
					end = len(hexstr)
				}
				fmt.Printf("%s\n", hexstr[i:end])
			}
			if dumpLen < len(data) {
				fmt.Printf("... (+%d bytes)\n", len(data)-dumpLen)
			}
		}

		if pcapWriter != nil {
			ci := gopacket.CaptureInfo{
				Timestamp:     time.Now(),
				CaptureLength: len(data),
				Length:        len(data),
			}
			if err := pcapWriter.WritePacket(ci, data); err != nil {
				log.Printf("write pcap: %v", err)
			}
		}

		printed++
		if *count > 0 && printed >= *count {
			break
		}
	}
}
