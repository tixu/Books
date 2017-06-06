package main

import (
	"context"
	"testing"

	pb "github.com/tixu/books/books/stub"
)

func TestCreateBook(t *testing.T) {

	s := server{file: "db_test.db"}
	s.init()
	book := &pb.Book{Author: "ken_follet", Title: "les Piliers de la terre"}

	rbook, err := s.CreateBook(context.Background(), &pb.CreateBookRequest{Book: book})
	if err != nil {
		t.Error(err)
	}
	if rbook.GetBook()[0].Id == 0 {
		t.Errorf("sequence not working")
	}
	s.close()
}
func TestDeleteBook(t *testing.T) {

	s := server{file: "db_test.db"}
	s.init()
	book := &pb.Book{Author: "ken_follet", Title: "les Piliers de la terre"}

	rbook, err := s.CreateBook(context.Background(), &pb.CreateBookRequest{Book: book})
	if err != nil {
		t.Error(err)
	}
	if rbook.GetBook()[0].Id == 0 {
		t.Errorf("sequence not working")
	}
	s.DeleteBook(context.Background(), &pb.DeleteBookRequest{Bookid: rbook.GetBook()[0].Id})
	s.close()
}
