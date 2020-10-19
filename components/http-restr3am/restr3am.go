package httprestr3am



import "net/http"
import "fmt"
import "errors"
import "bufio"


type ProxyListener struct {
	DataChannel chan []byte
  waitingHeader chan bool
}


//struct represent the proxystream
type ProxyStream struct {
	url         string
  authBasic   bool
	username    string
	password    string
	listeners map[*ProxyListener]bool
  io chan ioListenerMsg
  stop chan bool
  header http.Header

}

type ioListenerMsg interface {
  message () interface{}
}
type subscribeListener struct {
  listener *ProxyListener
}

type unsubscribeListener struct {
  listener *ProxyListener
}

type PayloadListeners struct {
  data []byte
}

type HeaderStream struct {
  header  http.Header
}


type ErrorStream struct {
  err error
}

type HandlerStream func  ()

func (s subscribeListener ) message () (interface {}) {
return s
}

func (u unsubscribeListener ) message () (interface {}) {
return u
}

func (p PayloadListeners ) message () (interface {}) {
return p
}

func (e ErrorStream ) message () (interface {}) {
return e
}

func (h HeaderStream ) message () (interface {}) {
return h
}


func NewProxyListener() *ProxyListener {
	listener := new(ProxyListener)

	listener.DataChannel = make(chan []byte)
  listener.waitingHeader=make(chan bool)
	return listener
}

//  create a New ProxyStream
func NewProxyStream(url string) (*ProxyStream) {

  proxy := new(ProxyStream)
	proxy.url = url
	proxy.listeners = make(map[*ProxyListener]bool)
  proxy.io = make(chan ioListenerMsg)
  proxy.stop=make(chan bool)
  proxy.header=nil
  proxy.username=""
  proxy.password=""
	return proxy

}

func (p *ProxyStream )SetBasicAuth (username, password string) {
  p.username=username
  p.password=password

}


func (p *ProxyStream )addListener (listener *ProxyListener) {
   msg:= subscribeListener{listener:listener}
   p.io <- msg
}

func (p *ProxyStream )removeListener (listener *ProxyListener) {
   msg:= unsubscribeListener{listener:listener}
   p.io <- msg
}


func (p *ProxyStream )writeData (listener *ProxyListener,payload PayloadListeners) {


defer func() {
       // recover from panic caused by writing to a closed channel
       if r := recover(); r != nil {

           fmt.Printf("write: error writing on channel: %v\n", listener.DataChannel)
           return
       }
   }()

   listener.DataChannel <- payload.data
}











func (p *ProxyStream )connectHTTPStream(url string) (*http.Response, error) {

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

 fmt.Printf("Resquest %s\n",url)
  if p.username != "" && p.password != "" {
		req.SetBasicAuth(p.username, p.password)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		err_str := fmt.Sprintf("Request failed (%s)", resp.Status)
		return nil,  errors.New(err_str)
	}



	return resp, nil;
}


func readChunkData(reader *bufio.Reader, size int) (buf []byte, err error) {
	buf = make([]byte, size)
	err = nil

	pos := 0
	for pos < size {
		var n int
		n, err = reader.Read(buf[pos:])
		if err != nil {
			return
		}

		pos += n
	}

	return
}

func (p *ProxyStream )HTTPStreamRequest () {

resp,err:=p.connectHTTPStream(p.url)

if err!=nil {
   //error in connection so we stop send the message
   msg:= ErrorStream{err:err}
   p.io <- msg
   if resp !=nil {
     resp.Body.Close()
   }
   //error we quit
   return
}

//we have the header we can attach and

  //= resp.Header
  //send message we have receive header
  fmt.Printf("Try send header from request\n")
  msg:= HeaderStream{header:resp.Header}
  p.io <- msg
  fmt.Printf("Header from request:OK\n")
  reader := bufio.NewReader(resp.Body)

  ChunkLoop:
  for {
    select {
      case <- p.stop:
      fmt.Printf("Explicit stopped\n")
      return


      default:


     		data, err := readChunkData(reader, 8192)
     		if err != nil {
          msg:= ErrorStream{err:err}
          p.io <- msg
     			break ChunkLoop
     		}

    		p.io <- PayloadListeners{data:data}

    }


  }


}

func (p *ProxyStream ) distpatch (handler HandlerStream ) {

  for {
		select {
		case data, ok := <- p.io:
			if ok {
        switch data.(type) {
        case subscribeListener:
          fmt.Printf("New subscription... %d listerners\n",len(p.listeners))
          p.listeners[data.(subscribeListener).listener] = true
          if (len(p.listeners) == 1) {
            p.stop=make(chan bool)
            go handler()
          }

        case unsubscribeListener:
          if (len(p.listeners) == 1) {
            p.header=nil
              go func () {


                defer func() {
                       // recover from panic caused by writing to a closed channel
                       if r := recover(); r != nil {

                           fmt.Printf("write: error writing on Stopped channel: %v\n")
                           return
                       }
                }()
								p.stop <- true // try to send


              }()

             fmt.Printf("Last listener stop the stream\n")


          }
          delete(p.listeners, data.(unsubscribeListener).listener)


        //case RequestEnded:
        case PayloadListeners:
          for listener, _ := range p.listeners {
            p.writeData(listener,data.(PayloadListeners))
          }

        case ErrorStream:
          fmt.Printf("Problem when open stream %s\n",data.(ErrorStream).err)
          for listener, _ := range p.listeners {
            close(listener.DataChannel)
          }




        case HeaderStream:
            p.header=data.(HeaderStream).header
            for listener, _ := range p.listeners {
              select {
              case listener.waitingHeader <- true: // try to send
              default: // or skip this frame
              }
            }
            //we can
        default:
          fmt.Printf("Unknown type message receive %t\n",data)
        }


      }
		}
	}

}

func (p *ProxyStream ) proxyStreamHandler (w http.ResponseWriter, r *http.Request) (int, error) {



  fmt.Printf("Server: client %s connected\n", r.RemoteAddr)


  flusher, ok := w.(http.Flusher)
  if !ok {
    fmt.Printf("Server: client %s could not be flushed",
      r.RemoteAddr)
    return 503,errors.New("Can't flush with you")
  }




listener:=NewProxyListener()

p.addListener(listener)

if p.header==nil {
  //we set the header if we already have it
   fmt.Printf("Waiting header from request\n")
select {
case <-listener.waitingHeader:
  break
case  _, ok := <-listener.DataChannel:
  if(!ok) {
    p.removeListener(listener)

    return http.StatusInternalServerError,nil
  }
}

}

header := w.Header()
for k, v := range p.header {
  header[k] = v
}

loop:
for {

  // wait for next chunk
  select {
    case data, ok := <-listener.DataChannel:
     if (ok) {
      _, err := w.Write(data)
      flusher.Flush()
      // check for client close
      if err != nil {
        fmt.Printf("Server: client %s failed (%s)\n",
          r.RemoteAddr, err)
          close(listener.DataChannel)
        break loop
      }
     } else {
			  p.removeListener(listener)
       return http.StatusInternalServerError,nil
     }




   }




}


    p.removeListener(listener)

    return http.StatusInternalServerError,nil


}



//implement ServeHTTP
func (p *ProxyStream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if status, err := p.proxyStreamHandler(w, r); err != nil {

        switch status {
        default:

            // Catch any other errors we haven't explicitly handled
            http.Error(w, err.Error(), http.StatusInternalServerError)
        }
      }
}


func (p *ProxyStream ) Handle (route string,handler HandlerStream ) {
   http.Handle(route, p)
   go p.distpatch(handler)

}
