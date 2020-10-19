package dlinkproxy



import "fmt"
import dlinkremotemqttconfig "dlink-remote-mqtt/components/config"
import httprestr3am "dlink-remote-mqtt/components/http-restr3am"



func Routing(config *dlinkremotemqttconfig.Config) {
	// check parameters

  for  _, prox := range config.Server.Proxy {


		proxy := httprestr3am.NewProxyStream(fmt.Sprintf("http://%s/video/mjpg.cgi", prox.Ip))
	  proxy.SetBasicAuth(prox.Username, prox.Password)
	  proxy.Handle(prox.UrlMjpeg,proxy.HTTPStreamRequest)

  }

}
