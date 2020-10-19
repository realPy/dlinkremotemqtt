package dlinkserver



import "net/http"
import "fmt"
import dlinkremotemqttconfig "dlink-remote-mqtt/components/config"
import dlinkapi "dlink-remote-mqtt/components/api"
import dlinkproxy "dlink-remote-mqtt/components/proxy"

type route interface {
     String() string
     Routing(interface{})
}



func StartServer(config *dlinkremotemqttconfig.Config) {
	// check parameters




  go func(){

     dlinkapi.Routing(config)

     dlinkproxy.Routing(config)

	   err2 := http.ListenAndServe(config.Server.BindServer, nil)
	   if err2 != nil {
		     fmt.Printf("Failed to start server: %s\n", err2)
	      }

    }()
}
