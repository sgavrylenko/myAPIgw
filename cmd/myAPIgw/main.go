package main

import (
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Route struct {
	Name    string `mapstructure:"name"`
	Context string `mapstructure:"context"`
	Target  string `mapstructure:"target"`
}

type GatewayConfig struct {
	ListenAddr string  `mapstructure:"listenAddr"`
	Routes     []Route `mapstructure:"routes"`
}

func main() {
	log.SetOutput(os.Stdout)

	//viper.AddConfigPath("./config")
	viper.SetConfigType("yaml")
	viper.SetConfigFile("./config/default.yml")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Error could not load configuration: %v", err)
	}

	viper.AutomaticEnv()

	gatewayConfig := &GatewayConfig{}

	err = viper.UnmarshalKey("gateway", gatewayConfig)
	if err != nil {
		panic(err)
	}
	log.Println("Initiliazing routes....")

	r := mux.NewRouter()
	for _, route := range gatewayConfig.Routes {
		proxy, err := NewProxy(route.Target)
		if err != nil {
			log.Panic(err)
		}

		log.Printf("Mapping '%v' | %v ---> %v", route.Name, route.Context, route.Target)
		r.HandleFunc(route.Context+"/{targetPath:.*}", NewHandler(proxy))
	}
	log.Printf("Started server on %v", gatewayConfig.ListenAddr)
	log.Fatal(http.ListenAndServe(gatewayConfig.ListenAddr, r))
}

func NewHandler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = mux.Vars(r)["targetPath"]
		log.Println("Request URL: ", r.URL.String())
		p.ServeHTTP(w, r)
	}
}

func NewProxy(targetUrl string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(targetUrl)
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ModifyResponse = func(response *http.Response) error {
		dumpResponse, err := httputil.DumpResponse(response, false)
		if err != nil {
			return err
		}
		log.Println("Response: \r\n", string(dumpResponse))
		return nil
	}
	return proxy, nil
}
