.PHONY: all clean player server

BIN_DIR := bin

all: player server

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

player: $(BIN_DIR)
	go build -o $(BIN_DIR)/player ./cmd/player

server: $(BIN_DIR)
	go build -o $(BIN_DIR)/server ./cmd/server

clean:
	rm -rf $(BIN_DIR)
