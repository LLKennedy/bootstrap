package main

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/digitalocean/godo"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	err := runscript()
	if err != nil {
		log.Fatalln(err)
	}
}

func runscript() error {
	// Create SSH keys
	root, _, rootpub, userpub, err := loadKeys()
	if err != nil {
		err = createKeys()
		if err != nil {
			log.Println("failed to create keys")
			return err
		}
		root, _, rootpub, _, err = loadKeys()
		if err != nil {
			log.Println("failed to load keys from files")
			return err
		}
	}
	log.Println("WARNING: This tool deletes ALL existing droplets and SSH keys the key can control, do not put your key here unless you're OK with that.")
	fmt.Printf("API key: ")
	apikey, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Printf("\n")
	if err != nil {
		log.Println("failed to read api key from user input")
		return err
	}
	client := godo.NewFromToken(string(apikey))
	ctx := context.Background()
	droplets, _, err := client.Droplets.List(ctx, &godo.ListOptions{})
	if err != nil {
		log.Println("failed to list droplets")
		return err
	}
	for _, droplet := range droplets {
		_, err = client.Droplets.Delete(ctx, droplet.ID)
		if err != nil {
			log.Println("failed to delete droplet")
			return err
		}
	}
	keys, _, err := client.Keys.List(ctx, &godo.ListOptions{})
	if err != nil {
		log.Println("failed to list keys")
		return err
	}
	for _, key := range keys {
		_, err = client.Keys.DeleteByID(ctx, key.ID)
		if err != nil {
			log.Println("failed to delete key")
			return err
		}
	}
	records, _, err := client.Domains.Records(ctx, "lukekennedy.net", &godo.ListOptions{})
	if err != nil {
		log.Println("failed to list domain records")
		return err
	}
	for _, record := range records {
		if record.Type == "A" || record.Type == "AAAA" {
			_, err = client.Domains.DeleteRecord(ctx, "lukekennedy.net", record.ID)
			if err != nil {
				log.Println("failed to delete domain record")
				return err
			}
		}
	}
	rootkey, _, err := client.Keys.Create(ctx, &godo.KeyCreateRequest{
		Name:      "website-root",
		PublicKey: string(rootpub),
	})
	if err != nil {
		log.Println("failed to create key")
		return err
	}
	droplet, _, err := client.Droplets.Create(ctx, &godo.DropletCreateRequest{
		Name:   "lukekennedynet",
		Region: "nyc3",
		Size:   "s-1vcpu-1gb",
		Image: godo.DropletCreateImage{
			Slug: "ubuntu-18-04-x64",
		},
		SSHKeys: []godo.DropletCreateSSHKey{
			{
				ID: rootkey.ID,
			},
		},
		IPv6: true,
	})
	if err != nil {
		log.Println("faled to create droplet")
		return err
	}
	log.Println("Waiting up to 1 minute for droplet creation...")
	deadline := time.Now().Add(time.Minute)
	ip, err := droplet.PrivateIPv4()
	for err == nil && ip == "" && deadline.After(time.Now()) {
		time.Sleep(5 * time.Second)
		ip, _ = droplet.PublicIPv4()
		droplet, _, err = client.Droplets.Get(ctx, droplet.ID)
	}
	if err != nil {
		log.Println("failed to get public IP for new droplet")
		return err
	}
	log.Printf("new IP: %s", ip)
	ip6, _ := droplet.PublicIPv6()
	newRecords := []*godo.DomainRecordEditRequest{
		&godo.DomainRecordEditRequest{
			Type: "A",
			Name: "@",

			Data: ip,
		},
		&godo.DomainRecordEditRequest{
			Type: "A",
			Name: "*",
			Data: ip,
		},
	}
	ip6Records := []*godo.DomainRecordEditRequest{
		&godo.DomainRecordEditRequest{
			Type: "AAAA",
			Name: "@",
			Data: ip6,
		},
		&godo.DomainRecordEditRequest{
			Type: "AAAA",
			Name: "*",
			Data: ip6,
		},
	}
	if ip6 != "" {
		newRecords = append(newRecords, ip6Records...)
	}
	for _, record := range newRecords {
		_, _, err = client.Domains.CreateRecord(ctx, "lukekennedy.net", record)
		if err != nil {
			log.Println("failed to create domain record")
			return err
		}
	}
	keyring := agent.NewKeyring()
	keyring.Add(agent.AddedKey{
		PrivateKey: root,
	})
	keyringSigners, err := keyring.Signers()
	if err != nil {
		log.Println("failed to generate keyring")
		return err
	}
	time.Sleep(5 * time.Second)
	sshclient, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", ip), &ssh.ClientConfig{
		Config: ssh.Config{},
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(keyringSigners...),
		},
		User:            "root",
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil },
	})
	if err != nil {
		log.Println("failed first ssh client creation attempt, sleeping before trying again")
		time.Sleep(5 * time.Second)
		sshclient, err = ssh.Dial("tcp", fmt.Sprintf("%s:22", ip), &ssh.ClientConfig{
			Config: ssh.Config{},
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(keyringSigners...),
			},
			User:            "root",
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil },
		})
		if err != nil {
			log.Println("failed to create ssh client")
			return err
		}
	}
	defer sshclient.Close()
	testSession, err := sshclient.NewSession()
	if err != nil {
		log.Printf("First session failed to open with error: %v", err)
		log.Println("Waiting 10s before retrying...")
		testSession, err = sshclient.NewSession()
		if err != nil {
			log.Println("couldn't establish ssh session")
			return err
		}
	}
	testSession.Close()
	// Put script commands here
	commands := []string{
		`adduser --disabled-password --gecos "" web`,
		`mkdir /home/web/.ssh`,
		fmt.Sprintf(`echo "%s" > /home/web/.ssh/authorized_keys`, userpub),
		`chown web:web /home/web/.ssh`,
		`chown web:web /home/web/.ssh/authorized_keys`,
		`usermod -aG sudo web`,
		`echo "web ALL=NOPASSWD:/sbin/reboot" >> /etc/sudoers.d/web`,
		`apt-get -y update`,
		`apt-get -y install software-properties-common`,
		`add-apt-repository -y universe`,
		`add-apt-repository -y ppa:certbot/certbot`,
		`apt-get -y update`,
		`apt-get -y install certbot`,
		`apt-get -y install python3-certbot-dns-digitalocean`,
		`mkdir /etc/digitaloceans`,
		fmt.Sprintf(`echo "dns_digitalocean_token = %s" >> /etc/digitaloceans/api.ini`, apikey),
		// `certbot certonly --dns-digitalocean --dns-digitalocean-credentials /etc/digitaloceans/api.ini --dns-digitalocean-propagation-seconds 60 -d lukekennedy.net -d *.lukekennedy.net -m mail@lukekennedy.net --noninteractive --agree-tos`,
		// `sed -i "s/^PermitRootLogin yes*$/PermitRootLogin no/" /etc/ssh/sshd_config`, // This locks root out of the server, this makes unrestricted root impossible without rebuilding from scratch
	}
	for _, command := range commands {
		err := runCommand(sshclient, command, nil)
		if err != nil {
			log.Println("failed to run ssh command")
			return err
		}
	}

	log.Println("success")
	return nil
}

func runCommand(sshclient *ssh.Client, command string, stdin []byte) error {
	sess, err := sshclient.NewSession()
	if err != nil {
		log.Println("failed to create ssh session")
		return err
	}
	defer sess.Close()
	//log.Printf("Running command: %s", command)
	var stdout, stderr []byte
	workers := sync.WaitGroup{}
	workers.Add(3)
	commandStart := sync.WaitGroup{}
	commandStart.Add(1)
	go func() {
		defer func() {
			recover()
			workers.Done()
		}()
		outPipe, _ := sess.StdoutPipe()
		commandStart.Wait()
		stdout, _ = ioutil.ReadAll(outPipe)
	}()
	go func() {
		defer func() {
			recover()
			workers.Done()
		}()
		errPipe, _ := sess.StdoutPipe()
		commandStart.Wait()
		stderr, _ = ioutil.ReadAll(errPipe)
	}()
	go func() {
		defer func() {
			recover()
			workers.Done()
		}()
		inPipe, _ := sess.StdinPipe()
		commandStart.Wait()
		for _, b := range stdin {
			inPipe.Write([]byte{b})
		}
		inPipe.Close()
	}()
	commandDone := sync.WaitGroup{}
	commandDone.Add(1)
	commandStart.Done()
	go func() {
		defer func() {
			recover()
			commandDone.Done()
		}()
		err = sess.Run(command)
	}()
	workers.Wait()
	commandDone.Wait()
	if err != nil {
		return fmt.Errorf("ssh error: %w with (stdout: %s) and (stderr: %s)", err, stdout, stderr)
	}
	log.Printf("%s", stdout)
	return nil
}

func createKeys() (err error) {
	err = genKey("root")
	if err != nil {
		return
	}
	err = genKey("user")
	return
}

func genKey(name string) error {
	keyName := fmt.Sprintf("%s_ecdsa.key", name)
	pubName := fmt.Sprintf("%s_ecdsa.pub", name)
	os.Remove(keyName)
	os.Remove(pubName)
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil
	}
	privFile, err := os.OpenFile(keyName, os.O_RDONLY|os.O_CREATE, os.ModeExclusive)
	if err != nil {
		return nil
	}
	defer privFile.Close()
	pubFile, err := os.OpenFile(pubName, os.O_RDWR|os.O_CREATE, os.ModeExclusive)
	if err != nil {
		return nil
	}
	defer pubFile.Close()
	privDer, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil
	}
	privBlock := pem.Block{
		Type:    "EC PRIVATE KEY",
		Headers: nil,
		Bytes:   privDer,
	}
	err = pem.Encode(privFile, &privBlock)
	if err != nil {
		return nil
	}
	sshPub, err := ssh.NewPublicKey(key.Public())
	if err != nil {
		return nil
	}
	pubBytes := ssh.MarshalAuthorizedKey(sshPub)
	_, err = pubFile.Write(pubBytes)
	if err != nil {
		return nil
	}
	return nil
}

func loadKeys() (root, user *ecdsa.PrivateKey, rootPub, userPub string, err error) {
	root, rootPub, err = loadKey("root")
	if err != nil {
		return
	}
	user, userPub, err = loadKey("user")
	return
}

func loadKey(name string) (key *ecdsa.PrivateKey, pub string, err error) {
	var privFile, pubFile []byte
	privFile, err = ioutil.ReadFile(fmt.Sprintf("%s_ecdsa.key", name))
	if err != nil {
		return
	}
	pubFile, err = ioutil.ReadFile(fmt.Sprintf("%s_ecdsa.pub", name))
	if err != nil {
		return
	}

	block, _ := pem.Decode(privFile)
	if block == nil {
		err = fmt.Errorf("no pem blocks found for %s key", name)
		return
	}
	var genericKey crypto.PrivateKey
	genericKey, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return
	}
	var ok bool
	key, ok = genericKey.(*ecdsa.PrivateKey)
	if !ok {
		err = fmt.Errorf("non-ecdsa private key for %s key", name)
		return
	}
	pub = string(pubFile)
	return
}
