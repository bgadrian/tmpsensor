package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/mkideal/cli"

	"github.com/d2r2/go-dht"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type argT struct {
	cli.Helper
	Port     int `cli:"port" usage:"web server port"  dft:"9080"`
	Interval int `cli:"interval" usage:"seconds interval between 2 sensor reads"  dft:"15"`
	Pin      int `cli:"pin" usage:"pin for the DHT22 sensor"  dft:"4"`
	Diff     int `cli:"diff" usage:"Maximum % difference between 2 reads"  dft:"20"`
}

func main() {
	cli.Run(new(argT), run)
}

func run(ctx *cli.Context) error {
	args := ctx.Argv().(*argT)

	sensorName := "sensor_"
	sensorDesc := "DHT22 sensor data."
	temperature, humidity := setupPrometheus(sensorName, sensorDesc)
	var lastTemp, lastHum *float32

	update := func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Panic when reading sensor: %s \n", r)
			}
		}()
		fmt.Printf("Reading data from pin %d...\n", args.Pin)
		celsius, humidityPerc, _, err := dht.ReadDHTxxWithRetry(dht.DHT22, args.Pin, false, 4)
		if err != nil {
			fmt.Printf("failed %s\n", err)
			return
		}

		if lastTemp != nil && diffIsTooHigh(lastTemp, celsius, args) {
			fmt.Printf("ignored data, scrwed temp: %f \n", celsius)
			return //the data is screwed
		}
		lastTemp = &celsius
		temperature.Set(float64(*lastTemp))

		if lastHum != nil && diffIsTooHigh(lastHum, humidityPerc, args) {
			fmt.Printf("ignored humidity data %f \n", humidityPerc)
			return //the data is screwed
		}
		lastHum = &humidityPerc
		humidity.Set(float64(*lastHum))
		fmt.Printf("got %fC and %f %%\n", celsius, humidityPerc)
	}

	ticker := time.NewTicker(time.Second * time.Duration(args.Interval))
	go func() {
		update()
		for range ticker.C {
			update()
		}
	}()

	runServer(args, ticker)
	return nil
}

func runServer(args *argT, ticker *time.Ticker) {
	server := setupWebServer(args.Port)
	fmt.Println("Press CTRL-C to exit")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	fmt.Println("Received the close signal ...")
	//close everything
	ticker.Stop()
	err := server.Close()
	if err != nil {
		fmt.Println(err)
	}
}

func diffIsTooHigh(lastTemp *float32, celsius float32, args *argT) bool {
	return math.Abs(float64(*lastTemp/celsius)*float64(100)) > float64(args.Diff)
}

func setupWebServer(port int) *http.Server {
	router := http.NewServeMux()
	router.Handle("/metrics", promhttp.Handler())
	portAsString := strconv.Itoa(port)
	server := &http.Server{
		Addr:         ":" + portAsString,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	fmt.Printf("Listening on http://localhost:%s/metrics\n", portAsString)

	go func() {
		httpError := server.ListenAndServe()
		if httpError != nil {
			log.Println("While serving HTTP: ", httpError)
		}
	}()

	return server
}

func setupPrometheus(sensorName string, sensorDesc string) (prometheus.Gauge, prometheus.Gauge) {
	temperature := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: sensorName + "temperature_celsius",
		Help: sensorDesc,
	})
	prometheus.MustRegister(temperature)
	humidity := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: sensorName + "humidity_percentage",
		Help: sensorDesc,
	})
	prometheus.MustRegister(humidity)
	return temperature, humidity
}
