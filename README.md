# DLINK REMOTEMQTT

## Introduction
This project was create to easyly integrate multiple dlink Webcam to homeassistant with functionality
 * This act as a proxy from cam to allow multiple device (specially for equipement
   with a few ressource and not able to stream to multiple client )
 * Automatically configure webcam cam on home-assistant
 * Use the PIR and the remote motion as an occupancy sensor
 * Keep the latest image with movement in memory


## How to compile

`# go get && go build -o dlinkremotemqtt`

## Configuration

Dlinkremomqtt need a use the same mosquitto broker instance that homeassistant use.

The binary is run in foreground and need configuration file like this:

```bash
server:
  brokerMQTT: "tcp://brokerip:1883"
  bindServer: ":8080"
  proxy:
   - ip: 192.168.1.254
     urlMjpeg: /room1.mjpg
     motionHA: true
     friendly_name: "room1"
     username: admin
     password: password
   - ip: 192.168.1.253
     urlMjpeg: /room2.mjpg
     motionHA: true
     friendly_name: "room2"
     username: admin
     password: password

```

The stream of the video stream is available at http://instance:8080/room1.mjpg  

Warn: No Authentification is available to consume the stream. If you want to expose this stream, please use a proxy (haproxy or nginx) with https and basic auth (or 2 way ssl ) to provide a good security.

## Camera control

You can also control each webcam with a MQTT message.  
Dlink mqtt use the NIPCA api.


### Move the camera from a relative position
Example to move the camera to the right
```
topic: /dlinkmqtt/relativePTZ
payload: { "name":"friendly_name", "pan": 0, "tilt": 20 }
```

### Set light mode
Available mode: auto, night, day
Example:
```
topic: /dlinkmqtt/SetLightMode
payload: { "name":"friendly_name", "mode": "day"}
```
### Go to the preset selection
Go to the saved position
Example:
```
topic: /dlinkmqtt/SetPTZPreset
payload: { "name":"friendly_name", "presetname": "home"}
```

### Save the preset selection
Save the current position as "preset name"
Example:
```
topic: /dlinkmqtt/SetCurrentPTZPreset
payload: { "name":"friendly_name", "presetname": "home"}
```
