package main


import (
	"flag"
	"fmt"
  dlinkremotemqttconfig "dlink-remote-mqtt/components/config"
	dlinkmqttdispatch "dlink-remote-mqtt/components/home-assistant"
	dlinkserver "dlink-remote-mqtt/components/server"
  //_ "net/http/pprof"
)





func main() {
	// check parameters

      var configPath string



      flag.StringVar(&configPath, "config", "./config.yml", "path to config file")

      flag.Parse()


      config, err := dlinkremotemqttconfig.LoadConfig(configPath)
      if err == nil {

      //start proxy in background

      dlinkserver.StartServer(config)

			dlinkmqttdispatch.ListenNotify(config)

			c:=make(chan bool)
			<-c

			} else {
				fmt.Printf("%v",err)
			}







}
