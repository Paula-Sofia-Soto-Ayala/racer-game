// import the necessary packages
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

var (
	host  = flag.String("host", "localhost", "server host")
	port  = flag.String("port", "9000", "server port")
	human = flag.Bool("human", true, "flag for human based client")
)

// client's main function
func main() {
	// check if the server address is provided as an argument
	flag.Parse()

	// get the server address from the argument
	server_address := *host + ":" + *port

	// if human client, wait for user input before continuing
	/* if *human {
		fmt.Print("Press enter to connect to the server...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	} */

	// connect to the server using net package (https://pkg.go.dev/net)
	conn, err := net.Dial("tcp", server_address)
	if err != nil {
		// print an error message and exit
		log.Fatal(err)
	}

	// defer closing the connection
	// defer conn.Close()

	// print a welcome message
	fmt.Println("Welcome to the racing game client!")

	// prompt the user for their name
	fmt.Print("Enter your name: ")

	// read a line from the standard input as the player name
	name, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		// print an error message and exit
		log.Fatal(err)
	}

	// write the player name to the connection
	_, err = fmt.Fprintln(conn, name)
	if err != nil {
		// print an error message and exit
		log.Fatal(err)
	}

	// create a channel to communicate between the main goroutine and the reader goroutine
	ch := make(chan struct{})

	// start a goroutine to read messages from the server and print them to the console
	go func() {
		// create a buffer to store the messages
		buf := make([]byte, 1024)

		// loop until the connection is closed
		for {
			// read from the connection
			n, err := conn.Read(buf)
			if err != nil {
				// check if the error is due to the connection being closed
				if err == io.EOF {
					// print a message and exit the loop
					fmt.Println("The connection is closed.")
					break
				} else {
					// print an error message and exit the loop
					log.Println(err)

					fmt.Print("Press enter to exit...")
					bufio.NewReader(os.Stdin).ReadBytes('\n')
					break
				}
			}

			// print the message to the console
			fmt.Print(string(buf[:n]))
		}

		// send a signal to the channel that the reader goroutine is done
		ch <- struct{}{}
	}()

	// start a loop to read input from the user and send it to the server
	for {
		// read a line from the standard input
		input, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			// print an error message and exit the loop
			log.Println(err)
			break
		}

		// write the input to the connection
		_, err = conn.Write([]byte(input))
		if err != nil {
			// print an error message and exit the loop
			log.Println(err)
			break
		}
	}

	// wait for the signal from the channel that the reader goroutine is done
	<-ch

	// exit the program
	os.Exit(0)
}
