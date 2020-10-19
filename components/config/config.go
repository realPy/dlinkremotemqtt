package dlinkremotemqttconfig

import (
   "gopkg.in/yaml.v2"
   "os"
)


type Proxy struct {
	Ip       string `yaml:"ip"`
	UrlMjpeg     string `yaml:"urlMjpeg"`
	Username     string `yaml:"username,omitempty"`
	Password     string `yaml:"password,omitempty"`
	MotionHA     bool `yaml:"motionHA"`
	Friendly_name string `yaml:"friendly_name,omitempty"`
}


type Config struct {

    Server struct {
      BrokerMQTT string `yaml:"brokerMQTT"`
      BindServer string `yaml:"bindServer,omitempty"`
        // Host is the local machine IP Address to bind the HTTP Server to
       Proxy [] Proxy `yaml:"proxy"`

    } `yaml:"server"`
}


func (p *Proxy) Name () string {

	if p.Friendly_name!="" {
		return p.Friendly_name
	}

   return p.Ip
}

func (config *Config) FindProxyByName(name string) (*Proxy) {

  for  _, prox := range config.Server.Proxy {
     if prox.Name() == name {
         return &prox
     }


  }
return nil
}
func LoadConfig(configPath string) (*Config, error) {



      config := &Config{}

  // Open config file
      file, err := os.Open(configPath)
      if err != nil {
          return nil,err
      }
      defer file.Close()

      // Init new YAML decode
      d := yaml.NewDecoder(file)
      d.SetStrict(true)
      // Start YAML decoding from file
      if err := d.Decode(&config); err != nil {

          return nil,err

      }
  return config,nil
}
