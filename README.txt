One Binary install of minimalistic orga tool:

Tech used:

server side rendering html
sqlite
async go backend

change port
go run main.go -port 1234


current ws challenge:

i think since i just render within projects the task board, the head is not loaded. -> this leads to the fact that htmx.ws.js is not loaded
ext  ws-connect="/projects/8ckGv7AfuPfRBpotlWt14w==/ws">
this is how it looks inside the html, which means it should work as intended, if it tries to open
<button hx-ws-connect="/ws" hx-trigger="click" hx-ws-send="Hello, server!" hx-ws-headers="Authorization: Bearer my-token">Connect</button>
this gives hope for headers

export LOG_LEVEL=debug
