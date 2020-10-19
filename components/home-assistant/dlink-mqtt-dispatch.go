package dlinkmqttdispatch



import "io"
import "bufio"
import "fmt"
import "strings"
import "net/http"
import "errors"
import "os"
import "os/signal"
import "io/ioutil"
import "time"
import "encoding/json"
import "github.com/eclipse/paho.mqtt.golang"
import guuid "github.com/google/uuid"
import dlinkremotemqttconfig "dlink-remote-mqtt/components/config"
import dlinkapi "dlink-remote-mqtt/components/api"





type dispatchAPI interface {
  name () string
}

type PTZRelative struct {
  Name string `json:"name"`
  Pan int  `json:"pan"`
  Tilt int `json:"tilt"`
}

func (p PTZRelative) name () string {

  return p.Name
}


type PTZPreset struct {
  Name string `json:"name"`
  PresetName string `json:"presetname"`
}

func (p PTZPreset) name () string {

  return p.Name
}

type LightMode struct {
  Name string `json:"name"`
  Mode string  `json:"mode"`
}

func (l LightMode) name () string {

  return l.Name
}


type HaDiscoverydevice struct {
  Name string `json:"name"`
  Sw_version string  `json:"sw_version"`
  Model string `json:"model"`
  Manufacturer string  `json:"manufacturer"`
  Identifiers []string `json:"identifiers"`
}

type HaDiscoveryBinaryDevice struct {
  Name string `json:"name"`
  Device_class string `json:"device_class"`
  State_topic string `json:"state_topic"`
  Unique_id string `json:"unique_id"`
  Device HaDiscoverydevice`json:"device"`
  lastcam *HaDiscoveryCam
}

type HaDiscoveryCam struct {
  Name string `json:"name"`
  Topic string `json:"topic"`
  Device HaDiscoverydevice`json:"device"`
}

type PayloadData interface {
    payload () interface{}

}

type MotionPayload struct {
   Topic string
   Data PayloadData
}

type configListenMQTT dlinkremotemqttconfig.Config

type BinaryPayload bool

func (b BinaryPayload) payload () interface {} {

  if b == true {
    return "ON"
  } else {
    return "OFF"
  }
}


type BytesPayload []byte

func (b BytesPayload) payload () interface {} {

  return []byte(b)
}


func getInfoFromDevice(ip string, username string, password string) (map[string]string,error){

  url := fmt.Sprintf("http://%s/common/info.cgi", ip)
  infodevice := make(map[string]string)

  req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return  nil,err
	}

	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil,err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		err_str := fmt.Sprintf("Request failed (%s)", resp.Status)
		return nil,errors.New(err_str)
	}

  reader := bufio.NewReader(resp.Body)

  for {

    line, err := reader.ReadSlice('\n')






    if err == io.EOF {
			break
		}
    if err != nil {
      return  nil,err
    }

      bl := strings.TrimRight(string(line), "\r\n");
      if (len(bl) > 0) {
         result := strings.Split(bl, "=")
         if(len(result)>1)  {
           infodevice[result[0]]=result[1]

         }

      }

  }
  resp.Body.Close()

  return infodevice,nil

}


func connect_stream_notify(url, username, password string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		err_str := fmt.Sprintf("Request failed (%s)", resp.Status)
		return nil, errors.New(err_str)
	}



	return resp, nil;
}

func GetSnapshot(url, username, password string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		err_str := fmt.Sprintf("Request failed (%s)", resp.Status)
		return nil, errors.New(err_str)
	}

  buffer, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil,err
	}
  resp.Body.Close()
	return buffer, nil;
}



func getHeader(resp http.Response) () {
	ct := resp.Header.Get("Content-Type")

  fmt.Printf("header: %s\n",ct)

}


func chunker(ip string,username string, password string,body io.ReadCloser,hainfo *HaDiscoveryBinaryDevice,payload chan MotionPayload) {


  reader := bufio.NewReader(body)
  defer body.Close()

  //ChunkLoop:
  for {

    line, err := reader.ReadSlice('\n')


    if err != nil {
      return
    }


      bl := strings.TrimRight(string(line), "\r\n");
      if (len(bl) > 0) {
         result := strings.Split(bl, "=")
         if(len(result)>1)  {

            if(result[0] == "pir" && result[1]=="on") {
              bytes,_:=GetSnapshot(fmt.Sprintf("http://%s/image/jpeg.cgi", ip), username, password)
              payload <- MotionPayload{Topic:hainfo.State_topic,Data:BinaryPayload(true)}


              payload <- MotionPayload{Topic:hainfo.lastcam.Topic,Data:BytesPayload(bytes)}

            }

            if(result[0] == "pir" && result[1]=="off") {
              payload <- MotionPayload{Topic:hainfo.State_topic,Data:BinaryPayload(false)}

            }

         }

      }




  }

}

func listen_notify_stream(ip string,username string, password string,hainfo *HaDiscoveryBinaryDevice,payload chan MotionPayload) {

    url_notify:=fmt.Sprintf("http://%s/config/notify_stream.cgi",ip)
    for {
      response, err := connect_stream_notify(url_notify, username,password)
      if err == nil  {
      //getHeader(*response)
      chunker(ip,username,password,response.Body,hainfo,payload)




      } else {
        fmt.Println("Error occured: ", err)
      }
    // we wait before reconnect
    time.Sleep(2 * time.Second)
    }

}






func sendDiscoveryHAMotion(client * mqtt.Client,hainfo *HaDiscoveryBinaryDevice) {

  jsonStr, _ := json.Marshal(hainfo)
  fmt.Printf("%s \n", jsonStr)
  token := (*client).Publish(fmt.Sprintf("homeassistant/binary_sensor/dlinkmotion_%s/config", hainfo.Name), 0, true, jsonStr)
  token.Wait()

}


func sendDiscoveryHALastCam(client * mqtt.Client,halastcam *HaDiscoveryCam) {

  jsonStr, _ := json.Marshal(halastcam)
  fmt.Printf("%s \n", jsonStr)
  token := (*client).Publish(fmt.Sprintf("homeassistant/camera/dlinklastcam_%s/config", halastcam.Name), 0, true, jsonStr)
  token.Wait()

}


func sendDestructHAMotion(client * mqtt.Client,hainfo *HaDiscoveryBinaryDevice) {
  token := (*client).Publish(fmt.Sprintf("homeassistant/binary_sensor/dlinkmotion_%s/config", hainfo.Name), 0, true, "")
  token.Wait()
}



func sendDestructHALastCam(client * mqtt.Client,hainfo *HaDiscoveryCam) {
  token := (*client).Publish(fmt.Sprintf("homeassistant/camera/dlinklastcam_%s/config", hainfo.Name), 0, true, "")
  token.Wait()
}

func (c configListenMQTT) dlinkmqttcommand(client mqtt.Client, msg mqtt.Message) {

  //switch r.Method
    var data interface{}
    var fn=func(data interface{},proxy *dlinkremotemqttconfig.Proxy)(error){return nil}
    data=nil
    fn=nil
    topic:=msg.Topic()
    switch topic {
    case "/dlinkmqtt/relativePTZ":
      data=&PTZRelative{}
      fn=func(data interface{},proxy *dlinkremotemqttconfig.Proxy)(error) {
        _,error:=dlinkapi.SetPTZRelativePosition (proxy.Ip, proxy.Username,proxy.Password,data.(*PTZRelative).Pan,data.(*PTZRelative).Tilt)
        return error
      }

    case "/dlinkmqtt/SetLightMode":
      data=&LightMode{}
      fn=func(data interface{},proxy *dlinkremotemqttconfig.Proxy)(error) {
        _,error:=dlinkapi.SetLightMode (proxy.Ip, proxy.Username,proxy.Password,data.(*LightMode).Mode)
        return error
      }

    case "/dlinkmqtt/SetPTZPreset":
      data=&PTZPreset{}
      fn=func(data interface{},proxy *dlinkremotemqttconfig.Proxy)(error) {
        _,error:=dlinkapi.GoPresetPosition (proxy.Ip, proxy.Username,proxy.Password,data.(*PTZPreset).PresetName)
        return error
      }

    case "/dlinkmqtt/SetCurrentPTZPreset":
      data=&PTZPreset{}
      fn=func(data interface{},proxy *dlinkremotemqttconfig.Proxy)(error) {
        _,error:=dlinkapi.SavePresetPosition (proxy.Ip, proxy.Username,proxy.Password,data.(*PTZPreset).PresetName)
        return error
      }

    default:
      fmt.Printf("Topic not found: %s\n", msg.Topic())
    }


    if data !=nil {

      if err:=json.Unmarshal(msg.Payload(), &data); err==nil {
        config:=dlinkremotemqttconfig.Config(c)

        if proxy:=config.FindProxyByName(data.(dispatchAPI).name()); proxy!=nil {

          if fn !=nil {
            fn(data,proxy)
          }

        } else {
           fmt.Printf("Error TOPIC %s: %s not found entity\n",topic,data.(dispatchAPI).name())
        }

      } else {
       fmt.Printf("Error TOPIC %s: %v\n",topic, err)
      }
    }



}



func ListenNotify(config *dlinkremotemqttconfig.Config){
  payload := make(chan MotionPayload)

  opts := mqtt.NewClientOptions()
  uuid := guuid.New()
  opts.SetClientID(fmt.Sprintf("dlinkmqtt_%s",uuid))
  opts.SetCleanSession(false)
  opts.AddBroker(config.Server.BrokerMQTT)


  opts.SetOnConnectHandler(func(client mqtt.Client){
    client.Subscribe("/dlinkmqtt/#", 0,func(client mqtt.Client, msg mqtt.Message) {
    c:=configListenMQTT(*config)
    c.dlinkmqttcommand(client,msg)})})
  client := mqtt.NewClient(opts)



  if token := client.Connect(); token.Wait() && token.Error() != nil {
    panic(token.Error())
 }


  var haDevices [](*HaDiscoveryBinaryDevice)

 for  _, prox := range config.Server.Proxy {
    infodevice := make(map[string]string)
    infodevice,err:=getInfoFromDevice(prox.Ip,prox.Username,prox.Password)
    if err==nil {
      hainfo := new(HaDiscoveryBinaryDevice)
      hainfo.Name=fmt.Sprintf("%s_motion", prox.Name())
      hainfo.State_topic=fmt.Sprintf("homeassistant/binary_sensor/dlinkmotion_%s/state", hainfo.Name)
      hainfo.Device_class="motion"
      var uniqueid=""
      var build=""
      var version=""
      var hw_version=""

      for key,data:=range infodevice {
        switch key {
        case "name":
          hainfo.Device.Name=fmt.Sprintf("%s_%s", data, prox.Name())
        case "product":
          hainfo.Device.Model=data
        case "build":
          build=data
        case "version":
          version=data
        case "hw_version":
          hw_version=data
        case "brand":
          hainfo.Device.Manufacturer=data
        case "macaddr":
          hainfo.Device.Identifiers=make([]string,1)
          uniqueid=strings.ToLower(strings.ReplaceAll(data,":",""))
          hainfo.Device.Identifiers[0]=fmt.Sprintf("dlink_0x%s",uniqueid)
        }

      }
      hainfo.Unique_id=fmt.Sprintf("0x%s_motion",uniqueid)
      hainfo.Device.Sw_version=fmt.Sprintf("%s_%s_%s",build,version,hw_version)
      haDevices=append(haDevices,hainfo)
      sendDiscoveryHAMotion(&client,hainfo)
      halastcam:= new(HaDiscoveryCam)
      halastcam.Name=fmt.Sprintf("%s_lastcam", prox.Name())
      halastcam.Topic=fmt.Sprintf("lastcam/dlinklastcam_%s", halastcam.Name)
      halastcam.Device=hainfo.Device
      hainfo.lastcam=halastcam
      //halastcam.Device_class="nil"
      sendDiscoveryHALastCam(&client,halastcam)
      go listen_notify_stream(prox.Ip,prox.Username,prox.Password,hainfo,payload)
    }

 }


 //install handler to cleanup HA
  c := make(chan os.Signal, 1)
  signal.Notify(c, os.Interrupt)
  go func(){

       <-c

     // sig is a ^C, handle it

     for _,hainfo:= range haDevices {
       sendDestructHAMotion(&client,hainfo)
       sendDestructHALastCam(&client,hainfo.lastcam)
     }

      os.Exit(0)
  }()




  for {

     motionpayload:= <-payload
     token := client.Publish(motionpayload.Topic, 0, false, motionpayload.Data.payload())
     if  token.Wait() && token.Error() != nil {
	    	fmt.Println(token.Error())
		    os.Exit(27)
    	}


  }
  defer client.Disconnect(250)

}
