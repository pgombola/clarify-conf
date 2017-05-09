package main

import (
	"flag"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type nodes struct {
	Nodes   []node
	Clarify clarify
}

type node struct {
	Hostname     string
	NetInterface string
	Address      string
	NomadPort    int
	Tools        string
}

type clarify struct {
	Install string
	Share   string
	User    string
}

func main() {
	cfg := flag.String("cfg", "nodes.yaml", "Nodes configuration to read")
	flag.Parse()

	n := &nodes{}
	data, err := ioutil.ReadFile(*cfg)
	if err != nil {
		fmt.Printf("Unable to read %s file.", *cfg)
	}
	if err := yaml.Unmarshal([]byte(data), &n); err != nil {
		fmt.Println(err)
	}
	for _, node := range n.Nodes {
		fmt.Println(node)
	}
	fmt.Println(n.Clarify)
}
