package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type config struct {
	Nodes   []*node  `yaml:"clarify-nodes"`
	Clarify *clarify `yaml:"clarify-common"`
}

type node struct {
	Hostname     string
	NetInterface string
	Address      string
	Tools        string
}

type clarify struct {
	Install   string
	Share     string
	User      string
	NomadPort int
}

type args struct {
	Args []string
}

func main() {
	cfg := flag.String("cfg", "nodes.yaml", "Nodes configuration to read")
	flag.Parse()

	config, err := parse(cfg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	node, err := config.findLocalNode()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	peers, _ := config.peers()

	cmd := &exec.Cmd{
		Dir:  config.Clarify.Install,
		Path: "jre/bin/java",
		Args: newArgs(config.Clarify, node, peers).Args,
	}

	if err := cmd.Run(); err != nil {
		fmt.Println(err)
	}
}

func parse(filename *string) (*config, error) {
	c := &config{}
	data, err := ioutil.ReadFile(*filename)
	if err != nil {
		return c, fmt.Errorf("unable to read %s", *filename)
	}
	if err := yaml.Unmarshal([]byte(data), &c); err != nil {
		return c, err
	}
	return c, nil
}

func (n *config) findLocalNode() (*node, error) {
	for _, n := range n.Nodes {
		if hostname, err := os.Hostname(); err != nil {
			return &node{}, err
		} else if n.Hostname == hostname {
			return n, nil
		}
	}
	return &node{}, errors.New("node not found")
}

func (n *config) peers() (string, error) {
	var peers = make([]string, 0)
	for _, n := range n.Nodes {
		if hostname, err := os.Hostname(); err != nil {
			return "", err
		} else if n.Hostname != hostname {
			peers = append(peers, n.Address)
		}
	}
	return strings.Join(peers, " "), nil
}

func newArgs(c *clarify, n *node, peers string) *args {
	args := &args{}
	args.user(c.User)
	args.toolsInstall(n.Tools)
	args.clarifyInstall(c.Install)
	args.clarifyShare(c.Share)
	args.netInterface(n.NetInterface)
	args.address(n.Address)
	args.nomad(c.NomadPort)
	args.hosts(peers)
	return args
}

func (a *args) user(user string) {
	a.Args = append(a.Args, "-user")
	a.Args = append(a.Args, user)
}

func (a *args) toolsInstall(dir string) {
	a.Args = append(a.Args, "-install")
	a.Args = append(a.Args, dir)
}

func (a *args) clarifyInstall(dir string) {
	a.Args = append(a.Args, "-clarify")
	a.Args = append(a.Args, dir)
}

func (a *args) clarifyShare(dir string) {
	a.Args = append(a.Args, "-share")
	a.Args = append(a.Args, dir)
}

func (a *args) netInterface(net string) {
	a.Args = append(a.Args, "-net")
	a.Args = append(a.Args, net)
}

func (a *args) address(address string) {
	if len(address) == 0 {
		return
	}
	a.Args = append(a.Args, "-address")
	a.Args = append(a.Args, address)
}

func (a *args) nomad(port int) {
	portStr := strconv.Itoa(port)
	a.Args = append(a.Args, "-nomad_port")
	a.Args = append(a.Args, portStr)
}

func (a *args) hosts(peers string) {
	a.Args = append(a.Args, "-hosts")
	a.Args = append(a.Args, peers)
}
