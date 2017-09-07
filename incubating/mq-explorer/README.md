
Docker for Mac
--------------

1. Install XQuartz.  Version 2.7.10 works, but V2.7.11 doesn't seem to.
2. Run XQuartz
3. Open the XQuartz "Preferences" menu, go to the "Security" tab and enable "Allow connections from network clients"
4. Add your IP address to the list of allowed hosts: `xhost + $(ipconfig getifaddr en0)`
5. Run MQ Explorer: `docker run -e DISPLAY=$(ipconfig getifaddr en0):0 -v /tmp/.X11-unix:/tmp/.X11-unix -u 0 -ti mq-explorer`

https://stackoverflow.com/questions/38686932/how-to-forward-docker-for-mac-to-x11

docker run -e DISPLAY=docker.for.mac.localhost:0 -v /tmp/.X11-unix:/tmp/.X11-unix -u 0 -ti mq-explorer
Use DISPLAY=docker.for.mac.localhost:0 ???
