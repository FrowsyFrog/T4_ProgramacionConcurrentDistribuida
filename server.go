package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type LinearRegression struct {
	slope     float64
	intercept float64
	isTrained bool
}

const (
	port = 8000
)

var (
	hostAddr string
	lr       LinearRegression
)

func (lr *LinearRegression) Fit(X, y []float64) {
	if len(X) != len(y) {
		panic("X and y must have the same length")
	}

	type PartialSums struct {
		sumX, sumY, sumXY, sumXSquare float64
	}

	partialSumsChan := make(chan PartialSums, len(X))
	var wg sync.WaitGroup
	wg.Add(len(X))

	for i := 0; i < len(X); i++ {
		go func(i int) {
			defer wg.Done()
			partialSumsChan <- PartialSums{
				sumX:       X[i],
				sumY:       y[i],
				sumXY:      X[i] * y[i],
				sumXSquare: X[i] * X[i],
			}
		}(i)
	}

	wg.Wait()
	close(partialSumsChan)

	var sumX, sumY, sumXY, sumXSquare float64

	for partial := range partialSumsChan {
		sumX += partial.sumX
		sumY += partial.sumY
		sumXY += partial.sumXY
		sumXSquare += partial.sumXSquare
	}

	n := float64(len(X))

	lr.slope = (n*sumXY - sumX*sumY) / (n*sumXSquare - sumX*sumX)
	lr.intercept = (sumY - lr.slope*sumX) / n

	lr.isTrained = true
}

func (lr *LinearRegression) Predict(X []float64) ([]float64, error) {
	if lr.isTrained {
		predictions := make([]float64, len(X))
		for i := range X {
			predictions[i] = lr.slope*X[i] + lr.intercept
		}
		return predictions, nil
	}
	return []float64{}, errors.New("El modelo requiere ser entrenado por lo menos una vez.")
}

func ReadDataset(url string) ([]float64, []float64) {

	var x []float64
	var y []float64

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error al hacer la solicitud HTTP: ", err)
		return x, y
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: no se pudo descargar el archivo CSV. Código de estado: ", resp.StatusCode)
		return x, y
	}

	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Error al leer el archivo CSV:", err)
		return x, y
	}

	for i, record := range records {

		// Omitir primera linea
		if i == 0 {
			continue
		}

		xVal, err := strconv.ParseFloat(record[0], 64)
		if err != nil {
			fmt.Println("Error al convertir a float64:", err)
			return x, y
		}

		yVal, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			fmt.Println("Error al convertir a float64:", err)
			return x, y
		}

		x = append(x, xVal)
		y = append(y, yVal)
	}

	return x, y
}

func initializeTraining() {
	X, y := ReadDataset("https://raw.githubusercontent.com/FrowsyFrog/T4_ProgramacionConcurrentDistribuida/main/train.csv")
	lr.Fit(X, y)
}

func main() {
	hostAddr = discoverIP()
	hostAddr = strings.TrimSpace(hostAddr)
	fmt.Println("Ejecutando con la IP:", hostAddr)

	registerServer()
}

func discoverIP() string {
	interfaces, _ := net.Interfaces()
	for _, iface := range interfaces {
		if strings.HasPrefix(iface.Name, "Ethernet") {
			addrs, _ := iface.Addrs()
			for _, addr := range addrs {
				switch t := addr.(type) {
				case *net.IPNet:
					if t.IP.To4() != nil {
						return t.IP.To4().String()
					}
				}
			}
		}
	}
	return "127.0.0.1"
}

func registerServer() {
	go initializeTraining()
	hostname := fmt.Sprintf("%s:%d", hostAddr, port)

	ls, _ := net.Listen("tcp", hostname)
	defer ls.Close()

	for {
		conn, _ := ls.Accept()
		go handleMessage(conn)
	}
}

func handleMessage(conn net.Conn) {
	defer conn.Close()

	for {
		msg, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			return
		}
		msg = strings.TrimSpace(msg)
		var X []float64
		json.Unmarshal([]byte(msg), &X)
		fmt.Println("Arreglo recibido:", X)

		result, err := lr.Predict(X)
		if err != nil {
			fmt.Fprintf(conn, "%s\n", err)
			continue
		}
		send(conn, result)
	}
}

func send(conn net.Conn, arr []float64) {
	bytesMsg, _ := json.Marshal(arr)
	fmt.Fprintln(conn, string(bytesMsg))
}
