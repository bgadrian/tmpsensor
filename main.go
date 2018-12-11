package main

import (
	"fmt"
	"log"
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
}

func main() {
	cli.Run(new(argT), run)
}

func run(ctx *cli.Context) error {
	args := ctx.Argv().(*argT)

	sensorName := "sensor_"
	sensorDesc := "DHT22 sensor data."
	temperature, humidity := setupPrometheus(sensorName, sensorDesc)

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

		temperature.Set(float64(celsius))
		humidity.Set(float64(humidityPerc))
		fmt.Printf("got %fC and %f %%\n", celsius, humidityPerc)
	}

	ticker := time.NewTicker(time.Second * time.Duration(args.Interval))
	go func() {
		update()
		for range ticker.C {
			update()
		}
	}()

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
	return nil
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
