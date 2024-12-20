package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/icholy/digest"
)

var reactFS embed.FS

var (
	packetID int
	seenIPs  = make(map[string]bool)
	mutex    sync.Mutex
	clients  = make(map[chan []byte]struct{})
)

func main() {

	distFS, err := fs.Sub(reactFS, "frontend/dist")
	if err != nil {
		log.Printf("Error accessing embedded frontend files: %v", err)
		fmt.Println("Press Enter to exit...")
		fmt.Scanln()
		log.Fatal(err)

	}

	// go func() {
	// 	if err := startSniffing(); err != nil {
	// 		log.Printf("Error starting sniffing: %v", err)
	// 		fmt.Println("Press Enter to exit...")
	// 		fmt.Scanln()
	// 	}
	// }()

	router := http.NewServeMux()
	router.Handle("/", http.FileServer(http.FS(distFS)))
	// router.HandleFunc("/events", eventsHandler)
	// router.HandleFunc("/clear", clearHandler)
	router.HandleFunc("/test", testHandler)

	server := http.Server{
		Addr:    ":7070",
		Handler: router,
	}

	log.Println("Starting HTTP server at http://localhost:7070")
	err = server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
		return
	}
	// var wg sync.WaitGroup
	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	if err := server.ListenAndServe(); err != nil {
	// 		log.Printf("Error starting HTTP server: %v", err)
	// 		fmt.Println("Press Enter to exit...")
	// 		fmt.Scanln()
	// 	}
	// }()

	// open.Start("http://localhost:7070")
	// wg.Wait()
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "OK"})
}

func clearHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight request
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	clearSeenIPs()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("IP list cleared"))
}

func startSniffing() error {
	interfaceName, err := findActiveInterface()
	if err != nil {
		return fmt.Errorf("error finding active interface: %v", err)
	}

	log.Printf("Starting packet sniffing on interface %s", interfaceName)

	handle, err := pcap.OpenLive(interfaceName, 1600, true, pcap.BlockForever)
	if err != nil {
		return fmt.Errorf("error opening device: %v", err)
	}
	defer handle.Close()

	var filter string = "udp port 14235 or udp port 8888 or udp port 12345 or udp port 11503 or udp port 60040 or udp port 45588 or udp port 9999"
	err = handle.SetBPFFilter(filter)
	if err != nil {
		return fmt.Errorf("error setting BPF filter: %v", err)
	}
	log.Printf("BPF filter set to: %s", filter)

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		log.Println("Packet received")
		handlePacket(packet)
	}
	return nil
}

func findActiveInterface() (string, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return "", err
	}

	var fallbackDevice string

	for _, device := range devices {
		log.Printf("Checking device: %s", device.Name)
		for _, address := range device.Addresses {
			if address.IP != nil && address.IP.To4() != nil && !address.IP.IsLoopback() {
				ip := address.IP.To4()
				log.Printf("Device %s has address: %s", device.Name, ip.String())
				if isPrivateIP(ip) {
					log.Printf("Selected interface: %s", device.Name)
					return device.Name, nil
				}

				fallbackDevice = device.Name
			}
		}
	}

	if fallbackDevice != "" {
		log.Printf("No preferred private IP found, using fallback interface: %s", fallbackDevice)
		return fallbackDevice, nil
	}

	return "", fmt.Errorf("no active interface found")
}

func isPrivateIP(ip net.IP) bool {
	privateIPBlocks := []*net.IPNet{
		{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
		{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},
		{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)},
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

func handlePacket(packet gopacket.Packet) {
	mutex.Lock()
	defer mutex.Unlock()

	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		log.Println("No IPv4 layer found in packet")
		return
	}
	ip, _ := ipLayer.(*layers.IPv4)
	log.Printf("IP packet: %s -> %s", ip.SrcIP, ip.DstIP)

	if seenIPs[ip.SrcIP.String()] {
		log.Println("IP already seen:", ip.SrcIP)
		return
	}
	seenIPs[ip.SrcIP.String()] = true
	packetID++

	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer == nil {
		log.Println("No UDP layer found in packet")
		return
	}
	udp, _ := udpLayer.(*layers.UDP)
	log.Printf("UDP packet: %d -> %d", udp.SrcPort, udp.DstPort)

	ethernetLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethernetLayer == nil {
		log.Println("No Ethernet layer found in packet")
		return
	}
	ethernet, _ := ethernetLayer.(*layers.Ethernet)
	log.Printf("Ethernet packet: %s -> %s", ethernet.SrcMAC, ethernet.DstMAC)

	data := map[string]interface{}{
		"id":      packetID,
		"ip_src":  ip.SrcIP.String(),
		"mac_src": ethernet.SrcMAC.String(),
		"port":    udp.DstPort,
	}

	if udp.DstPort == 14235 {
		minerInfo, err := getMinerInfo(ip.SrcIP.String())
		if err != nil {
			log.Printf("Error getting miner info: %v", err)
		} else {
			data["type"] = minerInfo["type"]
			data["rate_ideal"] = minerInfo["rate_ideal"]
		}
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshalling data: %v", err)
		return
	}

	// Stream data to the frontend
	for client := range clients {
		select {
		case client <- jsonData:
		case <-time.After(time.Second):
			close(client)
			delete(clients, client)
		}
	}

	log.Printf("New IP detected: %s on Port: %d with MAC: %s\n", ip.SrcIP.String(), udp.DstPort, ethernet.SrcMAC.String())
}

func getMinerInfo(ip string) (map[string]interface{}, error) {
	url := fmt.Sprintf("http://%s/cgi-bin/summary.cgi", ip)

	transport := &digest.Transport{
		Username: "root",
		Password: "root",
	}

	client := &http.Client{
		Transport: transport,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response code: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	info, ok := result["INFO"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("INFO field not found")
	}

	summary, ok := result["SUMMARY"].([]interface{})
	if !ok || len(summary) == 0 {
		return nil, fmt.Errorf("SUMMARY field not found or empty")
	}

	summaryMap, ok := summary[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid summary format")
	}

	rateIdeal, ok := summaryMap["rate_ideal"].(float64)
	if !ok {
		return nil, fmt.Errorf("rate_ideal field not found or invalid format")
	}

	minerInfo := map[string]interface{}{
		"type":       info["type"],
		"rate_ideal": rateIdeal / 1000,
	}

	return minerInfo, nil
}

func eventsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight request
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	client := make(chan []byte)
	clients[client] = struct{}{}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		select {
		case msg := <-client:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			close(client)
			delete(clients, client)
			return
		}
	}
}

func clearSeenIPs() {
	mutex.Lock()
	defer mutex.Unlock()
	packetID = 0
	seenIPs = make(map[string]bool)
	fmt.Println("\033[31mCleared the IP list.\033[0m")
}
