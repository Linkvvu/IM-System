package main

func main() {
	Server := NewServer("127.0.0.1", 1993)
	Server.Start()
}
