package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
	google_protobuf "github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	opentracing "github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	pb "github.com/tixu/books/books/stub"
	"google.golang.org/grpc"
)

const (
	//internal constante
	bookbucket = "BOOK"
	isbnbucket = "ISBN"

	// Our service name.
	serviceName = "Book-Service"

	// Host + port of our service.
	hostPort = "0.0.0.0:0"

	// Endpoint to send Zipkin spans to.
	zipkinHTTPEndpoint = "http://localhost:9411/api/v1/spans"

	// Debug mode.
	debug = false

	// same span can be set to true for RPC style spans (Zipkin V1) vs Node style (OpenTracing)
	sameSpan = true

	// make Tracer generate 128 bit traceID's for root spans.
	traceID128Bit = true
)

type server struct {
	file string
	DB   *bolt.DB
}

func main() {

	// Create our HTTP collector.
	collector, err := zipkin.NewHTTPCollector(zipkinHTTPEndpoint)
	if err != nil {
		fmt.Printf("unable to create Zipkin HTTP collector: %+v", err)
		os.Exit(-1)
	}

	// Create our recorder.
	recorder := zipkin.NewRecorder(collector, debug, hostPort, serviceName)

	// Create our tracer.
	tracer, err := zipkin.NewTracer(
		recorder,
		zipkin.ClientServerSameSpan(sameSpan),
		zipkin.TraceID128Bit(traceID128Bit),
	)
	if err != nil {
		fmt.Printf("unable to create Zipkin tracer: %+v", err)
		os.Exit(-1)
	}

	// Explicitly set our tracer to be the default tracer.
	opentracing.InitGlobalTracer(tracer)

	port := flag.Int("p", 8080, "port to listen to")
	file := flag.String("d", "person.db", "db files")
	flag.Parse()

	logrus.Infof("listening to port %d", *port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		logrus.Fatalf("could not listen to port %d: %v", *port, err)
	}

	s := grpc.NewServer(grpc.UnaryInterceptor(
		otgrpc.OpenTracingServerInterceptor(tracer)))
	server := server{file: *file}
	server.init()
	/*signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, os.Kill, os.Interrupt)
	go func() {
		for _ = range signalChan {
			server.close()
		}
	}() */
	fmt.Printf("server %s", server)
	pb.RegisterBooksServer(s, server)
	err = s.Serve(lis)
	if err != nil {
		logrus.Fatalf("could not serve: %v", err)
	}

}

func (s server) GetBook(ctx context.Context, br *pb.GetBookRequest) (*pb.BookReply, error) {

	books := []*pb.Book{}

	err := s.DB.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte(bookbucket))
		i := tx.Bucket([]byte(isbnbucket))
		book := pb.Book{}

		switch x := br.Criteria.(type) {
		case *pb.GetBookRequest_Bookid:
			{
				data := b.Get(itob(x.Bookid))
				if data != nil {
					err := proto.Unmarshal(data, &book)
					if err != nil {
						return err
					}
					books = append(books, &book)
					return nil
				}
				return nil

			}
		case *pb.GetBookRequest_Isbn:
			{
				logrus.Printf("looking for isbn %s", x.Isbn)
				id := i.Get([]byte(x.Isbn))

				data := b.Get(id)
				if data != nil {
					err := proto.Unmarshal(data, &book)
					if err != nil {
						return err
					}
					books = append(books, &book)
					return nil
				}
				return nil
			}
		default:
			return fmt.Errorf("not implemented")
		}
	})

	if err != nil {
		return nil, err
	}
	return &pb.BookReply{Book: books}, nil

}

func (s server) CreateBook(ctx context.Context, in *pb.CreateBookRequest) (*pb.BookReply, error) {
	books := []*pb.Book{}
	var book *pb.Book
	err := s.DB.Update(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte(bookbucket))
		i := tx.Bucket([]byte(isbnbucket))
		book = in.Book
		id, _ := b.NextSequence()
		book.Id = id

		data, err := proto.Marshal(in.Book)
		if err != nil {
			return err
		}

		b.Put(itob(book.Id), data)
		i.Put([]byte(book.GetIsbn13()), itob(book.Id))

		return nil
	})
	if err != nil {
		return nil, err
	}
	books = append(books, book)
	return &pb.BookReply{Book: books}, nil
}
func (s server) DeleteBook(ctx context.Context, in *pb.DeleteBookRequest) (*google_protobuf.Empty, error) {
	err := s.DB.Update(func(tx *bolt.Tx) error {
		book := pb.Book{}
		b := tx.Bucket([]byte(bookbucket))
		i := tx.Bucket([]byte(isbnbucket))
		data := b.Get(itob(in.Bookid))
		if data != nil {
			err := proto.Unmarshal(data, &book)
			if err != nil {
				return err
			}

			return nil
		}

		i.Delete([]byte(book.Isbn13))
		b.Delete(itob(in.Bookid))
		return nil

	})
	if err != nil {
		return nil, err
	}
	return &google_protobuf.Empty{}, nil

}

func (s *server) init() {
	logrus.Debug("initializing server")

	db, err := bolt.Open(s.file, 0644, nil)
	if err != nil {
		panic(err)
	}

	initDB(db)
	s.DB = db
	logrus.Debug("initializing database done")
}

func (s *server) close() {
	s.DB.Close()
	os.Remove(s.file)

}

func initDB(db *bolt.DB) {
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bookbucket))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		_, err = tx.CreateBucketIfNotExists([]byte(isbnbucket))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
}

// itob returns an 8-byte big endian representation of v.
func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}
