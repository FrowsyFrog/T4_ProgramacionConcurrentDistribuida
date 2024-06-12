package main

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"
)

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

func InitialTest(lr LinearRegression, X, y []float64) {
	start := time.Now()
	lr.Fit(X, y)
	elapsed := time.Since(start)

	fmt.Printf("Training time: %v\n", elapsed)
	fmt.Printf("Slope: %f\n", lr.slope)
	fmt.Printf("Intercept: %f\n", lr.intercept)

	newX := []float64{100}
	predictions := lr.Predict(newX)

	fmt.Println("Input: ", newX)
	fmt.Println("Predictions:", predictions)
}

func PerformanceTest(n int, lr LinearRegression, X, y []float64) []time.Duration {
	durations := make([]time.Duration, n)
	for i := 0; i < n; i++ {
		start := time.Now()
		lr.Fit(X, y)
		durations[i] = time.Since(start)
	}
	return durations
}

func Duration2Int(durations []time.Duration) []int {
	arr := make([]int, len(durations))
	for i, d := range durations {
		arr[i] = int(d)
	}
	return arr
}

func Int2Duration(ints []int) []time.Duration {
	arr := make([]time.Duration, len(ints))
	for i, val := range ints {
		arr[i] = time.Duration(val)
	}
	return arr
}

func getTop(n int, durations []time.Duration) []time.Duration {
	tempD2I := Duration2Int(durations)
	sort.Ints(tempD2I)
	return Int2Duration(tempD2I[:n])
}

func main() {
	X, y := ReadDataset("https://raw.githubusercontent.com/FrowsyFrog/T3_ProgramacionConcurrenteDistribuida/main/train.csv")
	var lr LinearRegression

	// InitialTest(lr, X, y)
	testTimes := PerformanceTest(1000, lr, X, y)
	topN := 10
	fmt.Println("Mejores", topN, "tiempos: ", getTop(topN, testTimes))
}
