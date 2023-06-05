# Define the binary names
SERVER_BINARY_NAME=server.out
CLIENT_BINARY_NAME=client.out
APP_NAME=racer

# Define the source files
SERVER_SOURCE=server.go
CLIENT_SOURCE=client.go

# Define the server address
SERVER_ADDRESS=127.0.0.1:3333

# Define the default rule that runs when you type 'make'
all:
	go mod init ${APP_NAME}
	go mod tidy
	make build-server
	make build-client

# Define the rule to build the server binary
build-server: $(SERVER_SOURCE)
	go build -o $(SERVER_BINARY_NAME) $(SERVER_SOURCE)

# Define the rule to build the client binary
build-client: $(CLIENT_SOURCE)
	go build -o $(CLIENT_BINARY_NAME) $(CLIENT_SOURCE)

# Define the rule to run the server
run-server: build-server
	./$(SERVER_BINARY_NAME) $(SERVER_ADDRESS)

# Define the rule to run the client
run-client: build-client
	./$(CLIENT_BINARY_NAME) $(SERVER_ADDRESS)

# Define the rule to clean up the binaries and other files
clean:
	go clean
	rm -rf *.out *.mod *.sum
