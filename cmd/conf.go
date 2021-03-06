package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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
}

type clarify struct {
	Install string
	Share   string
	User    string
	Tools   string
}

type args struct {
	Args []string
}

func main() {
	cfg := flag.String("cfg", "nodes.yml", "Nodes configuration to read")
	flag.Parse()

	config, err := parse(cfg)
	if err != nil {
		log.Fatal(err)
	}

	args, err := newArgs(config)
	if err != nil {
		log.Fatal(err)
	}
	if !strings.HasSuffix(config.Clarify.Install, string(os.PathSeparator)) {
		config.Clarify.Install = config.Clarify.Install + string(os.PathSeparator)
	}

	cmd := exec.Command(config.Clarify.Install+"jre/bin/java", args.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error executing command: %v\n", err)
	}
}

func parse(filename *string) (*config, error) {
	c := &config{}
	data, err := ioutil.ReadFile(*filename)
	if err != nil {
		return c, err
	}
	if err := yaml.Unmarshal(data, &c); err != nil {
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

func (n *config) peers() ([]string, error) {
	var peers = make([]string, 0)
	for _, n := range n.Nodes {
		if hostname, err := os.Hostname(); err != nil {
			return peers, err
		} else if n.Hostname != hostname {
			peers = append(peers, n.Address)
		}
	}
	return peers, nil
}

func newArgs(c *config) (*args, error) {
	args := &args{}
	node, err := c.findLocalNode()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Local node: {hostname=%s, net=%s, tools=%s, address=%s}\n",
		node.Hostname, node.NetInterface, c.Clarify.Tools, node.Address)

	peers, err := c.peers()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Peers: %s\n", peers)

	jar, err := findInstallerJar(c.Clarify.Install)
	if err != nil {
		return args, err
	}
	args.jar(jar)
	args.user(c.Clarify.User)
	args.toolsInstall(c.Clarify.Tools)
	args.clarifyInstall(c.Clarify.Install)
	args.clarifyShare(c.Clarify.Share)
	if err := args.netInterface(node.NetInterface, node.Address); err != nil {
		return args, err
	}
	args.address(node.Address)
	args.hosts(peers)
	return args, nil
}

func findInstallerJar(install string) (string, error) {
	if _, err := os.Stat(install); os.IsNotExist(err) {
		return "", errors.New("invalid install dir")
	}
	var jar string
	err := filepath.Walk(path.Join(install, "tools", "lib"),
		func(filepath string, info os.FileInfo, err error) error {
			matched, err := path.Match("clarify-service-installer-*", info.Name())
			if err != nil {
				return err
			}
			if matched {
				jar = filepath
			}
			return nil
		})
	if err != nil {
		return "", err
	} else if len(jar) == 0 {
		return "", errors.New("unable to locate service installer jar")
	}
	return jar, nil
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

func (a *args) netInterface(netInt, address string) error {
	i, _ := net.InterfaceByName(netInt)
	addrs, err := i.Addrs()
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		netIP, _, _ := net.ParseCIDR(addr.String())
		if netIP.To4().String() == address {
			a.Args = append(a.Args, "-net")
			a.Args = append(a.Args, netInt)
			return nil
		}
	}
	return fmt.Errorf("network interface (%s) is not bound to address (%s)", netInt, address)
}

func (a *args) address(address string) {
	a.Args = append(a.Args, "-address")
	a.Args = append(a.Args, address)
}

func (a *args) hosts(peers []string) {
	a.Args = append(a.Args, "-hosts")
	for _, peer := range peers {
		a.Args = append(a.Args, peer)
	}
}

func (a *args) jar(jar string) {
	a.Args = append(a.Args, "-jar")
	a.Args = append(a.Args, jar)
}

func (a *args) main() {
	a.Args = append(a.Args, "com.cleo.clarify.service.installer.Installer")
}
