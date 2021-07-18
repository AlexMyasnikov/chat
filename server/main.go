package main

import (
	"fmt"
	"github.com/ChuvashPeople/chat/services"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
	"sync"
)

func main() {
	fmt.Println("--- SERVER APP ---")
	listener, err := net.Listen("tcp", "localhost:5400")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	services.RegisterChatServiceServer(grpcServer, newServer())
	grpcServer.Serve(listener)
}

func newServer() *chatServiceServer {
	s := &chatServiceServer{
		channel: make(map[string][]chan *services.Message),
	}
	fmt.Println(s)
	return s
}

type chatServiceServer struct {
	services.UnimplementedChatServiceServer
	mu      sync.Mutex
	channel map[string][]chan *services.Message
}

func (s *chatServiceServer) JoinChannel(ch *services.Channel, msgStream services.ChatService_JoinChannelServer) error {
	msgChannel := make(chan *services.Message)
	s.channel[ch.Name] = append(s.channel[ch.Name], msgChannel)

	for {
		select {
		case <-msgStream.Context().Done():
			return nil
		case msg := <-msgChannel:
			fmt.Printf("GO ROUTINE (got message): %v \n", msg)
			msgStream.Send(msg)
		}
	}
}

func (s *chatServiceServer) SendMessage(msgStream services.ChatService_SendMessageServer) error {
	msg, err := msgStream.Recv()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}

	ack := services.MessageAck{Status: "SENT"}
	msgStream.SendAndClose(&ack)

	go func() {
		streams := s.channel[msg.Channel.Name]
		for _, msgChan := range streams {
			msgChan <- msg
		}
	}()
	return nil
}
