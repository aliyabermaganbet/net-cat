package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	clientsMap  = make(map[net.Conn]string) // map for connection nets and names
	messages    = make(chan message)        // messages
	inMessages  = make(chan message)
	outMessages = make(chan message) // leaving and joining messages
	mu          sync.Mutex
)

var (
	conn_host = "localhost"
	conn_type = "tcp"
	conn_port = "8080"
)

type message struct { // the message has its time sent, person who send, text string
	time string
	name string
	text string
}

func main() {
	listen, err := net.Listen(conn_type, conn_host+":"+conn_port) // network string and address
	fmt.Printf("Listening on the port:%s\n", conn_port)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	go DisplaytheText()
	history, err := os.Create("history.txt")
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	defer history.Close()
	for {
		connection, err := listen.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go handle(connection, history)
	}
}

func Logo() string {
	file, err := os.Open("logo.txt")
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	scanner := bufio.NewScanner(file)
	logo := ""
	for scanner.Scan() {
		logo += scanner.Text() + "\n"
	}
	return logo
}

func handle(connection net.Conn, history *os.File) {
	logo := Logo()
	connection.Write([]byte(logo))
	name := ""
	var err error
	for {
		connection.Write([]byte("[ENTER YOUR NAME]:"))
		reader := bufio.NewReader(connection)
		name, err = reader.ReadString('\n')
		if err != nil {
			log.Print(err)
			continue
		}
		if len(clientsMap) > 9 {
			connection.Write([]byte("Sorry, there is no available connection! Stay on the line until one becomes free!\n"))
			continue
		} else if name == "\n" {
			connection.Write([]byte("Sorry, it is necessary to enter your name!\n"))
			continue
		} else if !checkDuplicate(strings.TrimSpace(name)) {
			connection.Write([]byte("Sorry, this username is already taken!Please, enter another username!\n"))
			continue
		} else if !checkName(strings.TrimSpace(name)) {
			connection.Write([]byte("Invalid characters! Please, enter your name using correct letters!\n"))
			continue
		} else {
			break
		}
	}
	name = strings.TrimSpace(name)
	mu.Lock()
	clientsMap[connection] = name
	mu.Unlock()
	f, err := os.ReadFile(history.Name())
	connection.Write(f)
	fmt.Fprintf(connection, "\n[%v][%v]:", time.Now().Format("2006-1-2 15:4:5"), name)
	inMessages <- newMessage(name, "\nhas joined our chat...")
	history.Write([]byte("\n" + name + " has joined our chat..."))
	ms := bufio.NewScanner(connection)
	for ms.Scan() {
		fmt.Fprintf(connection, "[%s][%s]:", time.Now().Format("2006-1-2 15:4:5"), name)
		if ms.Text() == "" {
			continue
		}
		msg := newMessage(name, strings.TrimSpace(ms.Text()))
		history.Write([]byte(fmt.Sprintf("\n[%s][%s]:%s", time.Now().Format("2006-1-2 15:4:5"), msg.name, msg.text)))
		messages <- msg
	}
	mu.Lock()
	delete(clientsMap, connection)
	mu.Unlock()
	outMessages <- newMessage(name, "\nhas left our chat...")
	history.Write([]byte("\n" + name + " has left our chat..."))
	connection.Close()
}

func checkName(name string) bool {
	for _, v := range name {
		if (v > 0 && v < 47) || (v > 'Z' && v < 'a') || v > 'z' {
			return false
		}
	}
	return true
}

func checkDuplicate(name string) bool {
	for _, nm := range clientsMap {
		if name == nm {
			return false
		}
	}
	return true
}

func newMessage(name string, msg string) message {
	return message{
		name: name,
		text: msg,
	}
}

func DisplaytheText() {
	for {
		select {
		case msg := <-messages:
			mu.Lock()
			for conn, name := range clientsMap {
				if msg.name == name {
					continue
				}
				fmt.Fprintf(conn, "\n[%s][%s]:%s\n[%s][%s]:", time.Now().Format("2006-1-2 15:4:5"), msg.name, msg.text, time.Now().Format("2006-1-2 15:4:5"), name)
			}
			mu.Unlock()
		case inmsg := <-inMessages:
			mu.Lock()
			for conn, name := range clientsMap {
				if inmsg.name == name {
					continue
				}
				fmt.Fprintf(conn, "\n[%s][%s]:\n%s has joined our chat...\n[%s][%s]:", time.Now().Format("2006-1-2 15:4:5"), name, inmsg.name, time.Now().Format("2006-1-2 15:4:5"), name)
			}
			mu.Unlock()
		case outmsg := <-outMessages:
			mu.Lock()
			for conn, name := range clientsMap {
				fmt.Fprintf(conn, "\n[%s][%s]:\n%s has left our chat...\n[%s][%s]:", time.Now().Format("2006-1-2 15:4:5"), name, outmsg.name, time.Now().Format("2006-1-2 15:4:5"), name)
			}
			mu.Unlock()
		}
	}
}
