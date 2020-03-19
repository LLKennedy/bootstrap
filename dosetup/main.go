package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/digitalocean/godo"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	log.Println("WARNING: This tool deletes ALL existing droplets and SSH keys the key can control, do not put your key here unless you're OK with that.")
	fmt.Printf("API key: ")
	apikey, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Printf("\n")
	if err != nil {
		log.Fatalln(err)
	}
	client := godo.NewFromToken(string(apikey))
	ctx := context.Background()
	droplets, _, err := client.Droplets.List(ctx, &godo.ListOptions{})
	if err != nil {
		log.Fatalln(err)
	}
	for _, droplet := range droplets {
		_, err = client.Droplets.Delete(ctx, droplet.ID)
		if err != nil {
			log.Println(err)
		}
	}
	keys, _, err := client.Keys.List(ctx, &godo.ListOptions{})
	if err != nil {
		log.Fatalln(err)
	}
	for _, key := range keys {
		client.Keys.DeleteByID(ctx, key.ID)
	}

	rootkey, _, err := client.Keys.Create(ctx, &godo.KeyCreateRequest{
		Name:      "website-root",
		PublicKey: string(rootsshpublic.Marshal()),
	})
	if err != nil {
		log.Fatalln(err)
	}
	droplet, _, err := client.Droplets.Create(ctx, &godo.DropletCreateRequest{
		Name:   "website",
		Region: "nyc3",
		Size:   "s-1vcpu-1gb",
		Image: godo.DropletCreateImage{
			Slug: "ubuntu-14-04-x64",
		},
		SSHKeys: []godo.DropletCreateSSHKey{
			{
				ID: rootkey.ID,
			},
		},
	})
	droplet.PublicIPv4()
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("success")
}

func createKeys() (root *ecdsa.PrivateKey, rootPub string, user *ecdsa.PrivateKey, err error) {
	root, rootPub, err = genKey("root")
	if err != nil {
		return
	}
	user, _, err = genKey("user")
	return
}

func genKey(name string) (*ecdsa.PrivateKey, string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, "", err
	}
	privFile, err := os.Open(fmt.Sprintf("%s_ecdsa.key", name))
	if err != nil {
		return nil, "", err
	}
	defer privFile.Close()
	pubFile, err := os.Open(fmt.Sprintf("%s_ecdsa.pub", name))
	if err != nil {
		return nil, "", err
	}
	defer pubFile.Close()
	privDer, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, "", err
	}
	privBlock := pem.Block{
		Type:    "ECDSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDer,
	}
	err = pem.Encode(privFile, &privBlock)
	if err != nil {
		return nil, "", err
	}
	pubBytes, err := ssh.NewPublicKey(key)
	if err != nil {
		return nil, "", err
	}
	_, err = pubFile.Write(pubBytes.Marshal())
	if err != nil {
		return nil, "", err
	}
	return key, string(pubBytes.Marshal()), nil
}
