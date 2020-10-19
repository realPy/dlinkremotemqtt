package dlinkapi
import "fmt"
import "net/http"

import "io"
import "bufio"
import "strings"
import "errors"
import "strconv"

import dlinkremotemqttconfig "dlink-remote-mqtt/components/config"
import "encoding/json"


//type dlinkHandler func(http.ResponseWriter, *http.Request) (int, error)

type Apicamh struct {
   ip string
   username string
   password string
}


type NipcaPTZPosition struct {
  Pan int `json:"pan"`
  Tilt int `json:"tilt"`
}


type NipcaLightMode struct {
  Mode string `json:"mode"`
}

type NipcaPTZPreset struct {
  Name string `json:"name"`
  Act string `json:"act"`
}



// Our appHandler type will now satisify http.Handler
func (api *Apicamh) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if status, err := api.dlinkHTTPHandler(w, r); err != nil {
        // We could also log our errors centrally:
        // i.e. log.Printf("HTTP %d: %v", err)
        switch status {
        default:

            // Catch any other errors we haven't explicitly handled
            http.Error(w, err.Error(), http.StatusInternalServerError)
        }
      }
}

func (api *Apicamh ) dlinkHTTPHandler (w http.ResponseWriter, r *http.Request) (int, error) {


     if jsonStr,err:=GetPTZPosition(api.ip,api.username,api.password); err==nil {
       fmt.Fprintf(w, "%s",jsonStr)
       return http.StatusOK, err
     } else {
       return http.StatusInternalServerError,err
     }


     //http.Error(w, jsonStr, http.StatusInternalServerError)

    return http.StatusInternalServerError,nil


}


func getInfoByfunc (path string, ip string, username string, password string) (map[string]string,error){

  url := fmt.Sprintf("http://%s%s", ip,path)
  array := make(map[string]string)

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
		err_str := fmt.Sprintf("Request to %s failed (%s)",url, resp.Status)
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
           array[result[0]]=result[1]

         }

      }

  }

  resp.Body.Close()
  return array,nil

}



type NipcaResponseHandler func(map[string]string) (interface{},error)

func callNipcaFunc(url string, ip string, username string, password string,handler NipcaResponseHandler) (string, error){
  if info,error:=getInfoByfunc(url,ip,username,password); error == nil {
   if objJson,err:=handler(info); err==nil {
     str,err:=json.Marshal(objJson)
     return string(str),err
   }else {
     return "",err
   }


  }else {
    return "",error
  }

}


func GetPTZPosition (ip string, username string, password string) (string,error){

  return callNipcaFunc("/config/ptz_pos.cgi",ip,username,password,
  func(info map[string]string)(interface{},error) {
    pan,_:=strconv.Atoi(info["p"])
    tilt,_:=strconv.Atoi(info["t"])
    return NipcaPTZPosition{Pan:pan,Tilt:tilt},nil
  })

}

func SetPTZRelativePosition (ip string, username string, password string,pan int, tilt int) (string,error){
    return callNipcaFunc(fmt.Sprintf("/config/ptz_move_rel.cgi?p=%d&t=%d",pan,tilt),ip,username,password,
    func(info map[string]string)(interface{},error) {
      pan,_:=strconv.Atoi(info["p"])
      tilt,_:=strconv.Atoi(info["t"])
      return NipcaPTZPosition{Pan:pan,Tilt:tilt},nil
    })
}


func GetLightMode (ip string, username string, password string) (string,error){

  return callNipcaFunc(fmt.Sprintf("/config/icr.cgi"),ip,username,password,
  func(info map[string]string)(interface{},error) {
    return NipcaLightMode{Mode:info["mode"]},nil
  })

}


func SetLightMode (ip string, username string, password string,mode string) (string,error){
  switch mode {
  case "auto","day","night":

  return callNipcaFunc(fmt.Sprintf("/config/icr.cgi?mode=%s",mode),ip,username,password,
  func(info map[string]string)(interface{},error) {
    return NipcaLightMode{Mode:info["mode"]},nil
  })
  default :
    return "",errors.New("Unknown mode")
  }

return "",nil
}


func GoPresetPosition (ip string, username string, password string, presetName string) (string,error){

  return callNipcaFunc(fmt.Sprintf("/config/ptz_preset.cgi?name=%s&act=go",presetName),ip,username,password,
  func(info map[string]string)(interface{},error) {
    return NipcaPTZPreset{Name:info["name"],Act:info["act"]},nil
  })
}


func SavePresetPosition (ip string, username string, password string, presetName string) (string,error){

  return callNipcaFunc(fmt.Sprintf("/config/ptz_preset.cgi?name=%s&act=add",presetName),ip,username,password,
  func(info map[string]string)(interface{},error) {
    return NipcaPTZPreset{Name:info["name"],Act:info["act"]},nil
  })
}


/*

/cgi/ptdc.cgi?command=go_home
*/







func Routing(config *dlinkremotemqttconfig.Config) {
	// check parameters

  for  _, prox := range config.Server.Proxy {


     r:=fmt.Sprintf("/api/%s/", prox.Name())
     th := &Apicamh{ip:prox.Ip,username:prox.Username,password:prox.Password}
	   // start web server
     http.Handle(r, th)


  }
}
