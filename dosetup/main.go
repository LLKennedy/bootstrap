package main

import (
	"context"
	"fmt"
	"log"
	"syscall"

	"github.com/digitalocean/godo"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	fmt.Printf("API key: ")
	key, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Printf("\n")
	if err != nil {
		log.Println(err)
	}
	client := godo.NewFromToken(string(key))
	ctx := context.Background()
	droplets, _, err := client.Droplets.List(ctx, &godo.ListOptions{})
	if err != nil {
		log.Println(err)
	}
	for _, droplet := range droplets {
		_, err = client.Droplets.Delete(ctx, droplet.ID)
		if err != nil {
			log.Println(err)
		}
	}
	_, _, err = client.Droplets.Create(ctx, &godo.DropletCreateRequest{
		Name:   "test",
		Region: "nyc3",
		Size:   "s-1vcpu-1gb",
		Image: godo.DropletCreateImage{
			Slug: "ubuntu-14-04-x64",
		},
	})
	if err != nil {
		log.Println(err)
	}
	log.Println("success")
}
