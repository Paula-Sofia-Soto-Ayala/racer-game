package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// type Racer
type Racer struct {
	status      string
	name        string
	speed       float64
	max_speed   float64
	position    int
	lane        int
	current_lap int
}

// type Race
type Race struct {
	race_start_timer int
	current_lap      int
	lap_distance     int
	max_laps         int
	status           string
	lanes            []int
	racers           []Racer
	top_three        []Racer
}

// type Server
type Server struct {
	clients     []Client
	race        Race
	address     string
	max_players int
}

// type Client
type Client struct {
	conn    net.Conn // the connection to the client
	racer   Racer
	address string
	id      string
}

var (
	host      = flag.String("host", "localhost", "server host")
	port      = flag.String("port", "9000", "server port")
	numRacers = flag.Int("numRacers", 4, "number of racers")
	waitTime  = flag.Int("waitTime", 10, "wait time for the race to start")
	lapNumber = flag.Int("lapNumber", 10, "number of race laps")
)

// func start_server
func start_server() Server {
	flag.Parse()

	// create a localhost server using net package (https://pkg.go.dev/net)
	server := Server{}
	server.address = *host + ":" + *port

	// set the server's max_clients to a fixed value (e.g. 10)
	server.max_players = *numRacers

	// initialize a race with random number of laps between 20 and 40 and status not_started
	race := Race{}
	race.current_lap = 1
	race.max_laps = *lapNumber // rand.Intn(n) returns a random number between [0,n)
	race.status = "not_started"

	// set the race_start_timer to a fixed value (e.g. 10 seconds)
	race.race_start_timer = *waitTime

	// set the lap_distance to a fixed value (e.g. 1000 meters)
	race.lap_distance = 500

	// set the lanes to a list of numbers from [1, 6]
	race.lanes = make([]int, 6)
	for i := range race.lanes {
		race.lanes[i] = i + 1
	}

	// set the server's race to the initialized race
	server.race = race

	// wait for max_clients to connect or race_start_timer to expire
	// use a channel to communicate between the main goroutine and the listener goroutine
	// use a sync.WaitGroup to wait for all clients to be handled
	// use a mutex to protect the shared server state
	var wg sync.WaitGroup
	var mu sync.Mutex
	ch := make(chan net.Conn)

	// start a listener goroutine that accepts incoming connections and sends them to the channel
	fmt.Printf("Server started at %s\n", server.address)

	go func() {
		listener, err := net.Listen("tcp", server.address)

		if err != nil {
			log.Fatal(err)
		}

		defer listener.Close()

		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Println(err)
				continue
			}
			ch <- conn // send the connection to the channel
		}
	}()

	// start a timer goroutine that sends a signal to the channel after race_start_timer seconds
	go func() {
		time.Sleep(time.Duration(race.race_start_timer) * time.Second)
		ch <- nil // send a nil value to the channel to indicate timeout
	}()

	// loop until max_clients are connected or timeout occurs
	fmt.Printf("Waiting %ds before starting the race! 🚦\n", race.race_start_timer)
	for len(server.clients) < server.max_players {
		conn := <-ch // receive a value from the channel
		if conn == nil {
			// timeout occurred, break the loop
			break
		}

		// increment the wait group counter
		wg.Add(1)

		// handle the connection in a separate goroutine

		go func(c net.Conn, race *Race) {
			defer wg.Done()

			// read a line from the connection as the player name
			name, err := bufio.NewReader(conn).ReadString('\n')
			if err != nil {
				// print an error message and return
				log.Println(err)
				name = fmt.Sprintf("Player %d", rand.Intn(20)+1)
			}

			if len(name) <= 1 {
				name = fmt.Sprintf("Player %d", rand.Intn(20)+1)
			}

			// trim the newline character from the name
			name = strings.TrimSpace(name)

			// Prints that a player has joined
			fmt.Printf("%s just joined!\n", name)

			// create a new client with a unique id and a random racer
			client := Client{}
			client.address = c.RemoteAddr().String()
			client.id = uuid.New().String() // use github.com/google/uuid package to generate unique ids

			racer := Racer{}
			racer.name = name                        // use the client provided name
			racer.speed = 0                          // all cars start with a speed of 0 m/s
			racer.max_speed = rand.Float64()*10 + 55 // random speed between [55, 65) meters per second
			racer.position = 0                       // initial position is zero
			racer.current_lap = 1                    // initial lap is 1

			// assign a random lane to the racer from the available lanes
			lane_index := rand.Intn(len(race.lanes))
			racer.lane = race.lanes[lane_index]

			// remove the assigned lane from the available lanes
			// race.lanes = append(race.lanes[:lane_index], race.lanes[lane_index+1:]...)

			// set the racer status to waiting
			racer.status = "waiting"

			// assign the racer to the client
			client.racer = racer

			// assign the connection to the client
			client.conn = conn

			// lock the mutex before modifying the server state
			mu.Lock()

			// add the client to the server's client list
			server.clients = append(server.clients, client)

			// add the racer to the race's racer list
			server.race.racers = append(server.race.racers, racer)

			// unlock the mutex after modifying the server state
			mu.Unlock()

			// send a welcome message to the client
			fmt.Fprintf(c, "Welcome to the race, %s! Your speed is %.2f m/s and your lane is %d.\n", name, racer.speed, racer.lane)

		}(conn, &race) // pass the connection as an argument to the goroutine
	}

	// wait for all connection handler goroutines to finish
	wg.Wait()

	// fill the remaining slots in the race with CPU racers
	for len(server.race.racers) < server.max_players {
		var cpuName = fmt.Sprintf("CPU %d", len(server.race.racers)+1)
		fmt.Println(cpuName + " was added to the race 💻")

		racer := Racer{}
		racer.name = cpuName                     // use a simple naming scheme for CPU racers
		racer.speed = 0                          // all cars start with a speed of 0 m/s
		racer.max_speed = rand.Float64()*10 + 50 // random speed between [50, 60) meters per second
		racer.position = 0                       // initial position is zero
		racer.current_lap = 1                    // initial lap is 1

		// assign a random lane to the racer from the available lanes
		lane_index := rand.Intn(len(race.lanes))
		racer.lane = race.lanes[lane_index]

		// remove the assigned lane from the available lanes
		// race.lanes = append(race.lanes[:lane_index], race.lanes[lane_index+1:]...)

		// set the racer status to waiting
		racer.status = "waiting"

		// add the racer to the race's racer list
		server.race.racers = append(server.race.racers, racer)
	}

	// return the server object
	return server
}

// func start
func start() {
	// start the server and get the race object
	server := start_server()

	// start the race
	start_race(&server)

	// use a wait group to synchronize the main goroutine and the display/update goroutine
	var wg sync.WaitGroup

	// increment the wait group counter
	wg.Add(1)

	// start a display/update gorout
	// goroutine that runs while the race is ongoing
	go func() {
		defer wg.Done()

		// loop until the race status is complete
		for server.race.status != "complete" {

			// display the race status
			display_race_status(&server)

			// update the race state in-place
			update_race_status(&server)

			// sleep for a fixed interval (e.g. 1 second) to simulate time passing
			time.Sleep(time.Second)
		}
	}()

	// wait for the display/update goroutine to finish
	wg.Wait()

	// display the podium racers
	display_podium(server)

	// end the game and close the server
	end_game(server)
}

// func start_race
func start_race(server *Server) {

	// set the race status to ongoing
	server.race.status = "ongoing"

	// set the current lap to 1
	server.race.current_lap = 1

	// set the racer status to running for all racers
	for i := range server.race.racers {
		server.race.racers[i].status = "running"
	}

	// send a message to all clients that the race has started
	for _, client := range server.clients {
		if client.conn != nil {
			fmt.Fprintf(client.conn, "The race has started! Good luck!\n")
		}
	}
}

// func display_race_status
func display_race_status(server *Server) {
	// get the race object from the server

	// create a buffer to store the formatted output
	var buf bytes.Buffer

	// write the race status to the buffer
	fmt.Fprintf(&buf, "\nRace 🏁 status: %s\n", server.race.status)
	fmt.Fprintf(&buf, "Latest Lap: %d/%d\n", server.race.current_lap, server.race.max_laps)

	// loop through the racers and write their info to the buffer
	for _, racer := range server.race.racers {
		// draw the racer with a lane number, a car emoji, and a progress bar
		fmt.Fprintf(&buf, "%d 🏎️ [%s>%s]", racer.lane, strings.Repeat("=", racer.position/10), strings.Repeat(" ", (server.race.lap_distance-racer.position)/10))

		lap_display := fmt.Sprintf("Lap: %d/%d", racer.current_lap, server.race.max_laps)

		if racer.current_lap > server.race.max_laps {
			lap_display = "Finished! 🏁"
		}

		// write the racer's name, speed, and position to the buffer
		fmt.Fprintf(&buf, "%s - %s (%.2f m/s) %d/%dm\n", lap_display, racer.name, racer.speed, racer.position, server.race.lap_distance)
	}

	// send the buffer contents to each client
	for _, client := range server.clients {
		if client.conn != nil {
			fmt.Fprint(client.conn, buf.String())
		}
	}

	// print the buffer contents to the server console
	fmt.Print(buf.String())
}

// func update_race_status
func update_race_status(server *Server) {
	// loop through the racers and update their state
	for i := range server.race.racers {
		// get the racer object by reference
		racer := &server.race.racers[i]

		// check if the racer status is running
		if racer.status == "running" {
			// update the racer speed
			update_racer_speed(racer)

			// update the racer position
			update_racer_position(racer)

			// check if the racer position exceeds the lap distance
			if racer.position >= server.race.lap_distance {
				// update the racer lap
				update_racer_lap(racer, server)

				// check if the racer lap exceeds the max laps
				if racer.current_lap > server.race.max_laps {
					// update the racer status to finished
					update_racer_status(racer, server)
				}
			}

			// check if the racer can overtake another racer on the same lane
			if can_overtake(racer, &server.race) {
				// update the racer lane and notify its client
				update_racer_lane(racer, *server)
			}
		}
	}

	// update the race status and current lap based on the racers' state
	update_race_status_and_lap(&server.race)
}

// func update_racer_speed: updates the speed of a racer given the race conditions
// input: a pointer to a Racer object
// output: none (modifies the Racer object in place)
func update_racer_speed(racer *Racer) {
	// if the racer is on the first lap and has zero speed, accelerate quickly to its max speed
	if racer.current_lap == 1 && racer.speed == 0 {
		// increase the speed by a random factor between [0.5, 1.0) of the max speed
		racer.speed += (rand.Float64() + 0.5) * racer.max_speed
	} else {
		// otherwise, adjust the speed randomly by a small amount
		// increase or decrease the speed by a random factor between [-0.1, 0.1) of the max speed
		racer.speed += (rand.Float64()*0.3 - 0.1) * racer.speed

		// make sure the speed does not exceed the max speed or go below zero
		if racer.speed > racer.max_speed {
			racer.speed = racer.max_speed
		} else if racer.speed < 0 {
			racer.speed = 0
		}
	}
}

// func update_racer_position: updates the position of a racer based on its speed and time interval
// input: a pointer to a Racer object
// output: none (modifies the Racer object in place)
func update_racer_position(racer *Racer) {
	// assume the time interval is one second
	// increase the position by the racer's current speed (in meters per second)
	racer.position += int(racer.speed)
}

// func find_client_by_racer: finds the client that is associated with a given racer
// input: a Racer object and a Server object
// output: a pointer to a Client object or nil if no match is found
func find_client_by_racer(racer Racer, server Server) *Client {
	// loop through the clients in the race
	for _, client := range server.clients {
		// check if the client's racer name matches the given racer name
		if client.racer.name == racer.name {
			// return a pointer to the matching client
			return &client
		}
	}

	// return nil if no match is found
	return nil
}

// func update_racer_lap: updates the lap of a racer and resets its position to zero
// input: a pointer to a Racer object and a Race object
// output: none (modifies the Racer object in place)
func update_racer_lap(racer *Racer, server *Server) {
	// increment the lap by one
	racer.current_lap++

	// reset the position to zero
	racer.position = 0

	// send a message to the client (if any) that the racer has completed a lap
	if client := find_client_by_racer(*racer, *server); client != nil {
		if client.conn != nil {
			fmt.Fprintf(client.conn, "You have completed lap %d/%d.\n", racer.current_lap-1, server.race.max_laps)
		}
	}
}

// func update_racer_status: updates the status of a racer to finished and adds it to the top three list if applicable
// input: a pointer to a Racer object and a Race object
// output: none (modifies the Racer and Race objects in place)
func update_racer_status(racer *Racer, server *Server) {
	// set the racer status to finished
	racer.status = "finished"

	// send a message to the client (if any) that the racer has finished the race
	if client := find_client_by_racer(*racer, *server); client != nil {
		if client.conn != nil {
			fmt.Fprintf(client.conn, "You have finished the race!\n")
		}
	}

	// check if the top three list is full or not
	var cars_in_podium = len(server.race.top_three)
	if cars_in_podium < 3 {
		// add the racer to the top three list
		server.race.top_three = append(server.race.top_three, *racer)

		// send a message to the client (if any) that the racer has made it to the podium
		if client := find_client_by_racer(*racer, *server); client != nil {
			if client.conn != nil {
				fmt.Fprintf(client.conn, "You have made it to the podium! Your position was: %d\nCongratulations!\n", cars_in_podium+1)
			}
		}
	}
}

// func can_overtake: checks if a racer can overtake another racer on the same lane that is slower than it
// input: a Racer object and a Race object
// output: a boolean value indicating whether overtaking is possible or not
func can_overtake(racer *Racer, race *Race) bool {
	// loop through the other racers in the race
	for _, other := range race.racers {
		// check if the other racer is on the same lane as the racer
		if other.lane == racer.lane {
			// check if the other racer is slower than the racer and is within a certain distance (e.g. 10 meters)
			var can_overtake = math.Abs(float64(other.position-racer.position)) <= 10
			if other.speed < racer.speed && can_overtake {
				// return true if overtaking is possible
				return true
			}
		}
	}

	// return false as overtaking is not possible
	return false
}

// func update_racer_lane: updates the lane of a racer to an adjacent lane that is free of other racers
// input: a pointer to a Racer object and a Race object
// output: none (modifies the Racer object in place)
func update_racer_lane(racer *Racer, server Server) {
	// get the current lane of the racer
	current_lane := racer.lane

	// get the list of available lanes in the race
	available_lanes := server.race.lanes

	// create a list of adjacent lanes to the current lane
	adjacent_lanes := []int{}

	// loop through the available lanes
	for _, lane := range available_lanes {
		// check if the lane is adjacent to the current lane (i.e. one unit difference)
		var distance_to_lane = math.Abs(float64(lane - current_lane))
		if distance_to_lane <= 1.005 && distance_to_lane >= 0.995 {
			// add the lane to the adjacent lanes list
			adjacent_lanes = append(adjacent_lanes, lane)
		}
	}

	// check if there are any adjacent lanes available
	if len(adjacent_lanes) > 0 {
		// choose a random adjacent lane from the list
		new_lane := adjacent_lanes[rand.Intn(len(adjacent_lanes))]

		// update the racer's lane to the new lane
		racer.lane = new_lane

		// send a message to the client (if any) that the racer has changed lanes
		if client := find_client_by_racer(*racer, server); client != nil {
			if client.conn != nil {
				fmt.Fprintf(client.conn, "You have changed lanes from %d to %d.\n", current_lane, new_lane)
			}
		}
	}
}

// update the race status and current lap based on the racers' state
// input: a pointer to a Race object
// output: none (modifies the Race object in place)
func update_race_status_and_lap(race *Race) {
	// create a variable to store the maximum lap among the racers
	max_lap := 1

	// create a variable to store the number of racers who have finished the race
	finished_racers := 0

	// loop through the racers in the race
	for _, racer := range race.racers {
		// check if the racer's current lap is greater than the maximum lap
		if racer.current_lap > max_lap {
			// update the maximum lap to the racer's current lap
			max_lap = racer.current_lap
		}

		// check if the racer's status is finished
		if racer.status == "finished" {
			// increment the finished racers count by one
			finished_racers++
		}
	}

	// update the race's current lap to the maximum lap
	race.current_lap = max_lap

	// check if all racers have finished the race
	if finished_racers == len(race.racers) {
		// update the race's status to complete
		race.status = "complete"
	}
}

// display_podium: displays the podium with the top three racers, their names, and positions
// input: a Race object
// output: none (prints to the server console and sends to each client)
func display_podium(server Server) {
	// check if the race has a top three list
	if len(server.race.top_three) == 3 {
		// create a buffer to store the formatted output
		var buf bytes.Buffer

		// write a header message to the buffer
		fmt.Fprintln(&buf, "\nThe race is over! Here are the results:")

		// write a podium with ASCII graphics and emojis to the buffer
		fmt.Fprintln(&buf, "   🥇      🥈      🥉")
		fmt.Fprintln(&buf, "  / | \\   / | \\   / | \\")
		fmt.Fprintln(&buf, " /__|__\\ /__|__\\ /__|__\\")
		fmt.Fprintln(&buf, "|  ___  |  ___  |  ___  |")
		fmt.Fprintln(&buf, "| (___) | (___) | (___) |")
		fmt.Fprintln(&buf, "|_______|_______|_______|\n")

		// loop through the top three racers and write their names and positions to the buffer
		for i, racer := range server.race.top_three {
			// write the racer name and position to the buffer
			fmt.Fprintf(&buf, "%d. %s\n", i+1, racer.name)
		}

		// send the buffer contents to each client
		for _, client := range server.clients {
			if client.conn != nil {
				fmt.Fprint(client.conn, buf.String())
			}
		}

		// print the buffer contents to the server console
		fmt.Print(buf.String())
	} else {
		// print an error message if the race does not have a top three list
		fmt.Println("The race does not have a top three list.")
	}
}

// end_game: sends a message to each client to thank them for playing, and then ends the game and disconnects the clients
// input: a Server object
// output: none (closes the connections and exits the program)
func end_game(server Server) {
	// loop through the clients in the server
	for _, client := range server.clients {
		if client.conn != nil {
			// send a message to the client to thank them for playing
			fmt.Fprintf(client.conn, "Thank you for playing! Hope you had fun!\n")
			// close the client's connection
			client.conn.Close()
		}
	}

	// print a message to the server console to indicate the game is over
	fmt.Println("The game is over! Press ENTER to exit.")

	// wait for the user to press enter
	fmt.Scanln()

	// exit the program
	os.Exit(0)
}

// program's main function
func main() {
	// set the random seed to the current time
	rand.Seed(time.Now().UnixNano())

	// call the start function to run the game
	start()
}
