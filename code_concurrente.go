package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	port = 8000
)

var hostaddr string
var lr LinearRegression

var modelIsTrained bool = false

type LinearRegression struct {
	slope     float64
	intercept float64
}

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

	modelIsTrained = true
}

func (lr *LinearRegression) Predict(X []float64) []float64 {
	predictions := make([]float64, len(X))
	for i := range X {
		predictions[i] = lr.slope*X[i] + lr.intercept
	}
	return predictions
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

func descubrirIP() string {
	//interafz de red
	ifaces, _ := net.Interfaces()
	for _, i := range ifaces { //Interfaz de red
		if strings.HasPrefix(i.Name, "Ethernet") {
			//solo aquellos q son Ethernet
			addrs, _ := i.Addrs()
			for _, addr := range addrs {
				switch t := addr.(type) {
				case *net.IPNet:
					if t.IP.To4() != nil {
						return t.IP.To4().String() //retornamos la IP Ethernet V4
					}
				}
			}
		}
	}
	return "127.0.0.1"
}

func initializeTraining() {
	X, y := ReadDataset("https://raw.githubusercontent.com/FrowsyFrog/T4_ProgramacionConcurrentDistribuida/main/train.csv")
	lr.Fit(X, y)
}

func main() {
	hostaddr = descubrirIP()
	hostaddr = strings.TrimSpace(hostaddr)
	fmt.Println("Mi IP: ", hostaddr)

	go registerServer()

	//modo cliente
	//menú para conexión
	br := bufio.NewReader(os.Stdin)
	fmt.Print("Ingrese la IP de nodo remoto: ")
	remoteIP, _ := br.ReadString('\n')
	remoteIP = strings.TrimSpace(remoteIP)

	if remoteIP != "" {
		registerClient(remoteIP) //solicitar el enrrolamiento del nuevo nodo
	}
}

func registerServer() {
	go initializeTraining()
	hostname := fmt.Sprintf("%s:%d", hostaddr, port)
	ls, _ := net.Listen("tcp", hostname)
	defer ls.Close()
	//manejar las conexiones entrantes
	for {
		conn, _ := ls.Accept()
		go handleMessage(conn) //para soportar un alto volumen de conexiones???
	}
}

func handleMessage(conn net.Conn) {
	defer conn.Close()

	for {
		message, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			fmt.Println("Error al leer el mensaje del cliente:", err)
			continue
		}
		message = strings.TrimSpace(message)

		num, err := strconv.ParseFloat(message, 64)
		if err != nil {
			fmt.Fprintf(conn, "%s\n", "Se debe ingresar un número")
			continue
		}

		if modelIsTrained {
			results := lr.Predict([]float64{num})
			strResult := strconv.FormatFloat(results[0], 'f', 2, 64)
			fmt.Fprintf(conn, "%s\n", strResult)
			continue
		}

		fmt.Fprintf(conn, "%s\n", "El modelo no ha terminado de entrenar aún. Espera un momento antes de enviar un mensaje")
	}
}

func registerClient(remoteIP string) {
	remoteHost := fmt.Sprintf("%s:%d", remoteIP, port)
	println(remoteHost)
	//conectarme al nodo remoto
	conn, err := net.Dial("tcp", remoteHost)
	if err != nil {
		fmt.Println("Error al conectar al servidor:", err)
		return
	}
	defer conn.Close()

	// Enviar mensajes
	for {
		fmt.Print("Ingrese temperatura °C para enviar al servidor: ")
		br := bufio.NewReader(os.Stdin)
		message, _ := br.ReadString('\n')

		fmt.Fprintf(conn, "%s\n", message) //envío del mensaje
		//recibir resultado
		br = bufio.NewReader(conn)
		result, _ := br.ReadString('\n') //la bitácora de IPs
		fmt.Println("Respuesta temperatura en °F: ", result)
	}
}
