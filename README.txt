One Binary install of minimalistic orga tool:

Tech used:

server side rendering html
sqlite
async go backend

change port
go run main.go -port 1234


current ws challenge:

<button hx-ws-connect="/ws" hx-trigger="click" hx-ws-send="Hello, server!" hx-ws-headers="Authorization: Bearer my-token">Connect</button>
this gives hope for headers

export LOG_LEVEL=debug


on new project display empty task board

add create task func
add event task func for db, that can be stored
connect task func to button create

ws connect needs to be encrypted. do not allow otherwise.
need a self signed cert and ca setup...
