package main

import (
	"context"
	"grpc-chat/protos"
	"log"
	"net"
	"os"
	"sync"

	"google.golang.org/grpc"
	glog "google.golang.org/grpc/grpclog"
)

var grpcLog glog.LoggerV2

func init() {
	grpcLog = glog.NewLoggerV2(os.Stdout, os.Stdout, os.Stdout)
}

// Connection ...
type Connection struct {
	stream protos.Chat_ConnectServer
	name   string
	active bool
	err    chan error
}

// Server ...
type Server struct {
	Connection []*Connection
}

// Connect ...
func (s *Server) Connect(user *protos.User, stream protos.Chat_ConnectServer) error {
	conn := &Connection{
		stream: stream,
		name:   user.Name,
		active: true,
		err:    make(chan error),
	}
	s.Connection = append(s.Connection, conn)
	return <-conn.err
}

// SendMessage ...
func (s *Server) SendMessage(ctx context.Context, msg *protos.Message) (*protos.Close, error) {
	wait := sync.WaitGroup{}
	done := make(chan int)

	for _, conn := range s.Connection {
		log.Println(conn.name)
		wait.Add(1)

		go func(msg *protos.Message, conn *Connection) {
			defer wait.Done()

			if conn.active {
				err := conn.stream.Send(msg)
				grpcLog.Infof("Sending message %v to user %v", msg.User.Name, conn.name)

				if err != nil {
					grpcLog.Errorf("Errors with stream %v: %v", conn.stream, err)
					conn.active = false
					conn.err <- err
				}
			}
		}(msg, conn)
	}

	go func() {
		wait.Wait()
		close(done)
	}()

	<-done
	return &protos.Close{}, nil
}

// UsersInfo ...
func (s *Server) UsersInfo(ctx context.Context, req *protos.InfoRequest) (*protos.InfoResponse, error) {
	response := protos.InfoResponse{}
	for _, conn := range s.Connection {
		if conn.name == req.User.Name {
			continue
		}
		response.Info += conn.name
		response.Info += " "
	}
	return &response, nil
}

func main() {
	var connections []*Connection

	server := &Server{connections}

	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("error creating the server %v", err)
	}

	grpcLog.Info("Starting server at port :8080")

	protos.RegisterChatServer(grpcServer, server)
	grpcServer.Serve(listener)
}
