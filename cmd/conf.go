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
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Command returned error:\n%v\n", err)
	}
	fmt.Printf("Command output:\n%v\n", string(out))
}

func parse(filename *string) (*config, error) {
	c := &config{}
	data, err := ioutil.ReadFile(*filename)
	if err != nil {
		return c, fmt.Errorf("unable to read %s", *filename)
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

func (n *config) peers() (string, error) {
	var peers = make([]string, 0)
	for _, n := range n.Nodes {
		if hostname, err := os.Hostname(); err != nil {
			return "", err
		} else if n.Hostname != hostname {
			ips, err := net.LookupIP(n.Hostname)
			if err != nil {
				return "", err
			}
			peers = append(peers, ips[0].To4().String())
		}
	}
	return strings.Join(peers, " "), nil
}

func newArgs(c *config) (*args, error) {
	args := &args{}
	node, err := c.findLocalNode()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Local node: {hostname=%s, net=%s, tools=%s}\n", node.Hostname, node.NetInterface, node.Tools)

	peers, err := c.peers()
	if err != nil {
		log.Fatal(err)
	}
	jar, err := findInstallerJar(c.Clarify.Install)
	if err != nil {
		return args, err
	}
	args.jar(jar)
	args.user(c.Clarify.User)
	args.toolsInstall(node.Tools)
	args.clarifyInstall(c.Clarify.Install)
	args.clarifyShare(c.Clarify.Share)
	args.netInterface(node.NetInterface)
	args.address(node.Address)
	args.nomad(c.Clarify.NomadPort)
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
	a.Args = append(a.Args, fmt.Sprintf("-user %s", user))
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
	a.Args = append(a.Args, "-nomad.port")
	a.Args = append(a.Args, portStr)
}

func (a *args) hosts(peers string) {
	a.Args = append(a.Args, "-hosts")
	a.Args = append(a.Args, peers)
}

func (a *args) jar(jar string) {
	a.Args = append(a.Args, "-jar")
	a.Args = append(a.Args, jar)
}

func (a *args) main() {
	a.Args = append(a.Args, "com.cleo.clarify.service.installer.Installer")
}