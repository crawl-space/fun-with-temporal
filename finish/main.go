package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"go.temporal.io/sdk/client"
)

func main() {
	// Create the client object just once per process
	c, err := client.NewClient(client.Options{})
	if err != nil {
		log.Fatalln("unable to create Temporal client", err)
	}
	defer c.Close()

	token, _ := base64.StdEncoding.DecodeString(os.Args[1])
	err = c.CompleteActivity(context.TODO(), []byte(token), os.Args[2], nil)
	fmt.Println(err)
}
