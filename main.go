package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
)

func parseSeeds(seeds string) []string {
	if seeds == "" {
		return []string{}
	}
	return strings.Split(seeds, ",")
}

func main() {
	port := flag.String("port", "", "Node port")
	seeds := flag.String("seeds", "", "Other known nodes (comma-separated)")
	miner := flag.String("miner", "", "Miner address")
	flag.Parse()

	if *port == "" {
		log.Panic("No port specified. Use --port")
	}

	// 将port作为nodeID
	nodeID := *port

	seedNodes := parseSeeds(*seeds)
	if len(seedNodes) > 0 {
		KnownNodes = seedNodes
	}

	fmt.Printf("KnownNodes after parse: %v\n", KnownNodes)

	// 传入nodeID给StartServer，以便后续在NewBlockchain中使用nodeID生成独立db文件
	StartServer(nodeID, *miner)
}
