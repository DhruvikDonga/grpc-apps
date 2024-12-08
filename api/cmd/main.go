package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"

	pb "github.com/DhruvikDonga/grpc-apps/api/messages"
	"google.golang.org/grpc"
)

type Message struct {
	Room       string `json:"room"`
	ClientName string `json:"client_name"`
	Message    string `json:"message"`
}

type MessageServer struct {
	pb.UnimplementedMessageServiceServer // Embed the required struct
	messages                             []Message
	mu                                   sync.Mutex
}

var (
	startServer2 = false
	isK8s        = false
	namespace    = ""
	cluster      = ""
	svcname      = ""
)

func init() {
	if os.Getenv("IS_K8S") == "true" {
		isK8s = true
		if os.Getenv("SVCNAME") != "" {
			svcname = os.Getenv("SVCNAME")
		}
		if os.Getenv("CLUSTER") != "" {
			cluster = os.Getenv("CLUSTER")
		}
		if os.Getenv("NAMESPACE") != "" {
			namespace = os.Getenv("NAMESPACE")
		}
	}
}

func main() {

	m := &MessageServer{
		UnimplementedMessageServiceServer: pb.UnimplementedMessageServiceServer{}, // Embed the required type

		messages: make([]Message, 0),
		mu:       sync.Mutex{},
	}
	go func() {
		//start grpc server
		fmt.Println("Start GRPC server")
		listener, err := net.Listen("tcp", "0.0.0.0:9091")
		if startServer2 {
			listener, err = net.Listen("tcp", "0.0.0.0:9092")
		}
		if err != nil {
			panic("error building server: " + err.Error())
		}
		s := grpc.NewServer()
		pb.RegisterMessageServiceServer(s, m)

		if err := s.Serve(listener); err != nil {
			panic("error building server: " + err.Error())
		}
	}()

	http.HandleFunc("/add", m.handleAddMessage)
	http.HandleFunc("/messages", m.handleGetMessages)

	// Start the server
	fmt.Println("Start  server")

	if startServer2 {
		http.ListenAndServe("127.0.0.1:8082", nil)
	} else {
		http.ListenAndServe(":8081", nil)
	}

}

// handleAddMessage handles the POST request to add a new message.
func (m *MessageServer) handleAddMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	log.Println("ADD CALLED")

	var msg Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	m.messages = append(m.messages, msg)
	m.mu.Unlock()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleGetMessages handles the GET request to retrieve all messages.
func (m *MessageServer) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	log.Println("GET CALLED")
	res := []Message{}

	// dial to server all servers:= 9091,9092
	servers := GetPODIPs()
	for _, server := range servers {
		conn, err := grpc.Dial(server, grpc.WithInsecure())

		defer conn.Close()

		if err != nil {
			log.Println("Error connecting to gRPC server: ", err.Error())
		}
		client := pb.NewMessageServiceClient(conn)
		stream, err := client.GetAllMessages(context.Background(), &pb.Empty{})
		if err != nil {
			log.Println(err) // dont use panic in your real project
			break
		}

		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				// Gracefully exit the loop when the server closes the stream
				log.Println("Stream closed by server", server)
				break
			}
			if err != nil {
				// Log and handle unexpected errors
				log.Printf("Error receiving from stream: %v", err)
				break
			}
			// Serialize the response to JSON for logging
			res = append(res, Message{
				Room:       resp.Room,
				ClientName: resp.ClientName,
				Message:    resp.Message,
			})

		}

	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// GetAllMessages(*Empty, MessageService_GetAllMessagesServer) error
func (m *MessageServer) GetAllMessages(req *pb.Empty, srv pb.MessageService_GetAllMessagesServer) error {
	log.Println("Fetch data streaming", m.messages)
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, message := range m.messages {

		resp := pb.Message{
			Room:       message.Room,
			ClientName: message.ClientName,
			Message:    message.Message,
		}

		if err := srv.Send(&resp); err != nil {
			log.Println("error generating response")
			return err
		}
	}

	return nil
}

func GetPODIPs() []string {
	if isK8s {
		ips := []string{}
		serviceName := svcname + "." + namespace + ".svc." + "cluster" + ".local"

		addrs, err := net.LookupHost(serviceName)
		if err != nil {
			log.Printf("Error resolving DNS for service %s: %v", serviceName, err)
			return ips
		}

		for _, addr := range addrs {
			ips = append(ips, addr+":9091")
		}

		log.Printf("Resolved pod IPs: %v", ips)

		return ips
	} else {
		return []string{"0.0.0.0:9091", "0.0.0.0:9092"}
	}
}

//APP1
//normal:- 8081
//grpc:- 8091

//APP2
//normal:- 8082
//grpc:- 8092
