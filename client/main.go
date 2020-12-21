package main

import (
	"bufio"
	"fmt"
	"grpc-chat/protos"
	"os"

	"log"
	"sync"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var client protos.ChatClient
var wait *sync.WaitGroup

func init() {
	wait = &sync.WaitGroup{}
}

func connect(user *protos.User) error {
	var streamerror error

	stream, err := client.Connect(context.Background(), &protos.User{
		Name:   user.Name,
		Active: true,
	})

	if err != nil {
		return fmt.Errorf("connection failed: %v", err)
	}

	wait.Add(1)
	go func(str protos.Chat_ConnectClient) {
		defer wait.Done()

		for {
			msg, err := str.Recv()
			if err != nil {
				streamerror = fmt.Errorf("Error reading message: %v", err)
				break
			}

			fmt.Printf("%v : %s\n", msg.User.Name, msg.Message)
		}
	}(stream)

	return streamerror
}

func main() {
	done := make(chan int)

	fmt.Print("username: ")
	var name string
	fmt.Scan(&name)
	fmt.Println("type INFO to view all users")

	user := &protos.User{Name: name}
	user.Active = true

	conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Couldnt connect to service: %v", err)
	}

	client = protos.NewChatClient(conn)

	connect(user)

	wait.Add(1)
	go func() {
		defer wait.Done()

		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			msg := &protos.Message{
				User:    user,
				Message: scanner.Text(),
			}

			if msg.Message == "INFO" {
				req := &protos.InfoRequest{User: user}
				response, err := client.UsersInfo(context.Background(), req)
				if err != nil {
					fmt.Printf("Error Info Message: %v", err)
					break
				}
				fmt.Println(response.Info)
			} else {
				_, err := client.SendMessage(context.Background(), msg)
				if err != nil {
					fmt.Printf("Error Sending Message: %v", err)
					break
				}
			}
		}
	}()

	go func() {
		wait.Wait()
		close(done)
	}()

	<-done

}
