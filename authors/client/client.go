package main

import (
	"flag"
	"fmt"
	"log"

	"context"

	pb "github.com/tixu/books/authors/stub"
	"google.golang.org/grpc"
)

func main() {
	backend := flag.String("b", "localhost:8080", "address of the say backend")
	flag.Parse()

	conn, err := grpc.Dial(*backend, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect to %s: %v", *backend, err)
	}
	defer conn.Close()

	client := pb.NewAuthorClient(conn)
	resp, err := client.GetAuthor(context.Background(), &pb.AuthorRequest{Id: "sdlkslm"})

	if err != nil {
		panic(err)
	}
	fmt.Printf("response %+v", resp)
}
