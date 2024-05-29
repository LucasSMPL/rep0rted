package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

//go:embed all:frontend/dist
var reactFS embed.FS

var (
	packetID int
	seenIPs  = make(map[string]bool)
	mutex    sync.Mutex
)

func main() {
	// Serve the React frontend
	distFS, err := fs.Sub(reactFS, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}

	router := http.NewServeMux()
	router.Handle("/", http.FileServer(http.FS(distFS)))
	router.HandleFunc("/scan", scanHandler)
	router.HandleFunc("/clear", clearHandler)

	server := http.Server{
		Addr:    ":7070",
		Handler: router,
	}

	log.Println("Starting HTTP server at http://localhost:7070 ...")
	log.Fatal(server.ListenAndServe())
}

func scanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	go startSniffing()

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Sniffing started"))
}

func clearHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	clearSeenIPs()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("IP list cleared"))
}

func startSniffing() {
	handle, err := pcap.OpenLive("en0", 1600, true, pcap.BlockForever) // This Could Be Trouble Some - Need to find how to dynamically select 'interface'. en0 will only work on WiFi.
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	var filter string = "udp and (port 14235 or port 8888 or port 12345)"
	err = handle.SetBPFFilter(filter)
	if err != nil {
		log.Fatal(err)
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		handlePacket(packet)
	}
}

func handlePacket(packet gopacket.Packet) {
	mutex.Lock()
	defer mutex.Unlock()

	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		return
	}
	ip, _ := ipLayer.(*layers.IPv4)

	if seenIPs[ip.SrcIP.String()] {
		return
	}
	seenIPs[ip.SrcIP.String()] = true
	packetID++

	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer == nil {
		return
	}
	udp, _ := udpLayer.(*layers.UDP)

	ethernetLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethernetLayer == nil {
		return
	}
	ethernet, _ := ethernetLayer.(*layers.Ethernet)

	data := map[string]interface{}{
		"id":      packetID,
		"ip_src":  ip.SrcIP.String(),
		"mac_src": ethernet.SrcMAC.String(),
		"port":    udp.DstPort,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshalling data: %v", err)
		return
	}

	// Send data to the frontend
	resp, err := http.Post("http://localhost:7070/data", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error sending data: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("New IP detected: %s on Port: %d with MAC: %s\n", ip.SrcIP.String(), udp.DstPort, ethernet.SrcMAC.String())
}

func clearSeenIPs() {
	mutex.Lock()
	defer mutex.Unlock()
	packetID = 0
	seenIPs = make(map[string]bool)
	fmt.Println("Cleared the IP list.")
}
