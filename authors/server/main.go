package main

import (
	"flag"
	"fmt"
	"net"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	pb "github.com/tixu/books/authors/stub"
	"google.golang.org/grpc"
)

type server struct {
	Books map[string]*pb.Author
}

func main() {
	port := flag.Int("p", 8080, "port to listen to")
	flag.Parse()

	logrus.Infof("listening to port %d", *port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		logrus.Fatalf("could not listen to port %d: %v", *port, err)
	}

	s := grpc.NewServer()
	server := server{}
	fmt.Printf("server %s", server)
	pb.RegisterAuthorServer(s, server)
	err = s.Serve(lis)
	if err != nil {
		logrus.Fatalf("could not serve: %v", err)
	}

}

func (s server) GetAuthor(ctx context.Context, br *pb.AuthorRequest) (*pb.Author, error) {
	response := pb.Author{Firstname: "Ken", Lastname: "Follet", Year: 1973, Country: "England"}
	return &response, nil
}
