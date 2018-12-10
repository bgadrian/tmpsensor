package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/d2r2/go-dht"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	//TODO put them as flags
	port := "9080"
	intervalSeconds := 30
	dataPin := 7
	sensorName := "sensor_"
	sensorDesc := "DHT22 sensor data."

	temperature, humidity := setupPrometheus(sensorName, sensorDesc)

	update := func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Panic when reading sensor: %s \n", r)
			}
		}()
		fmt.Printf("Reading data from pin %d...\n", dataPin)
		celsius, humidityPerc, _, err := dht.ReadDHTxxWithRetry(dht.DHT22, dataPin, false, 4)
		if err != nil {
			fmt.Printf("failed %s\n", err)
			return
		}

		temperature.Set(float64(celsius))
		humidity.Set(float64(humidityPerc))
		fmt.Printf("got %fC and %f%%\n", celsius, humidity)
	}

	ticker := time.NewTicker(time.Second * time.Duration(intervalSeconds))
	go func() {
		update()
		for range ticker.C {
			update()
		}
	}()

	server := setupWebServer(port)

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

func setupWebServer(port string) *http.Server {
	router := http.NewServeMux()
	router.Handle("/metrics", promhttp.Handler())
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	fmt.Printf("Listening on http://localhost:%s/metrics\n", port)

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
