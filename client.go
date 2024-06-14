package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

var (
	hostAddr string
)

func readArray(br *bufio.Reader) []float64 {
	fmt.Print("Ingrese arreglo: ")
	strArr, _ := br.ReadString('\n')
	strArr = strings.TrimSpace(strArr)

	arr := []float64{}
	for _, strNum := range strings.Split(strArr, " ") {
		num, _ := strconv.ParseFloat(strNum, 64)
		arr = append(arr, num)
	}
	return arr
}

func main() {
	br := bufio.NewReader(os.Stdin)

	fmt.Print("Ingrese dirección: ")
	hostAddr, _ := br.ReadString('\n')
	hostAddr = strings.TrimSpace(hostAddr)

	conn, err := net.Dial("tcp", hostAddr)

	if err != nil {
		fmt.Println("ERROR: No se pudo establecer conexión con el servidor.")
		os.Exit(1)
	}

	handle(conn)
}

func handle(conn net.Conn) {
	defer conn.Close()

	br := bufio.NewReader(os.Stdin)
	for {
		X := readArray(br)
		send(conn, X)

		pred := getPredictions(conn)
		fmt.Println("Predicciones: ", pred)
	}
}

func send(conn net.Conn, arr []float64) {
	bytesMsg, _ := json.Marshal(arr)
	fmt.Fprintln(conn, string(bytesMsg))
}

func getPredictions(conn net.Conn) []float64 {
	msg, _ := bufio.NewReader(conn).ReadString('\n')
	msg = strings.TrimSpace(msg)
	var y []float64
	json.Unmarshal([]byte(msg), &y)
	return y
}
