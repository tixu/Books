package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"context"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	opentracing "github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	pb "github.com/tixu/books/books/stub"

	"google.golang.org/grpc"
)

const (

	// Our service name.
	serviceName = "cli"

	// Host + port of our service.
	hostPort = "0.0.0.0:0"

	// Endpoint to send Zipkin spans to.
	zipkinHTTPEndpoint = "http://localhost:9411/api/v1/spans"

	// Debug mode.
	debug = false

	// Base endpoint of our SVC1 service.
	svc1Endpoint = "localhost:8080"

	// same span can be set to true for RPC style spans (Zipkin V1) vs Node style (OpenTracing)
	sameSpan = true

	// make Tracer generate 128 bit traceID's for root spans.
	traceID128Bit = true
)

func main() {

	// Create our HTTP collector.
	collector, err := zipkin.NewHTTPCollector(zipkinHTTPEndpoint)
	if err != nil {
		fmt.Printf("unable to create Zipkin HTTP collector: %+v", err)
		os.Exit(-1)
	}

	// Create our recorder.
	recorder := zipkin.NewRecorder(collector, debug, hostPort, "Book")

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

	backend := flag.String("b", svc1Endpoint, "address of the say backend")

	flag.Parse()
	conn, err := grpc.Dial(*backend, grpc.WithInsecure(), grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(tracer)))
	if err != nil {
		log.Fatalf("could not connect to %s: %v", *backend, err)
	}
	defer conn.Close()

	client := pb.NewBooksClient(conn)
	book := pb.Book{Author: "ken_follet", Title: "les Piliers de la terre", Isbn13: "978-2221110829"}
	span := opentracing.StartSpan("Book")
	// Call the Concat Method
	span.LogEvent("Call Create")
	ctx := opentracing.ContextWithSpan(context.Background(), span)
	resp, _ := client.CreateBook(ctx, &pb.CreateBookRequest{Book: &book})
	span.LogEvent("Call Get ID")
	reqID := pb.GetBookRequest_Bookid{Bookid: 1}
	req := pb.GetBookRequest{Criteria: &reqID}
	span.LogEvent("Call GET ISBN")
	_, err = client.GetBook(ctx, &req)
	reqISBN := pb.GetBookRequest_Isbn{Isbn: "978-2221110829"}
	req2 := pb.GetBookRequest{Criteria: &reqISBN}

	respe, err := client.GetBook(ctx, &req2)
	fmt.Printf("result %+v", respe)
	span.LogEvent("DELETE ISBN")
	client.DeleteBook(ctx, &pb.DeleteBookRequest{Bookid: resp.Book[0].Id})
	span.Finish()

	if err != nil {
		panic(err)
	}
	collector.Close()
}
