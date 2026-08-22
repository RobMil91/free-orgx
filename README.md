# Minimalist Orga Tool

A lightweight, self-contained organization tool built for speed, simplicity, and zero internet reliance. Everything runs from a **single binary** with zero external network dependencies, making it perfect for hosting locally on lightweight hardware like a **Raspberry Pi**.

---

## About the Project

This started as a fun experimental playground to explore the **htmx** and server-driven events dynamically update UI components without heavy JavaScript frameworks or complex client-side state management.

---

## Tech Stack & Architecture

* **Language/Backend:** Async Go (Golang)
* **Frontend:** Server-Side Rendered (SSR) HTML driven by `htmx`
* **Database:** SQLite (Embedded, single-file storage)
* **Deploy Model:** Single static binary
* **Connectivity:** **100% offline-first** — zero online dependencies or external API calls

---

## Features

* **Single Binary Deployment:** No external runtime, database server setup, or asset directories required.
* **Low Footprint:** Runs seamlessly on low-power devices, including Raspberry Pi (3, 4, Zero 2 W, etc.).
* **Core Orga Capabilities:** Handles essential organizer tasks with ultra-fast page interactions and minimal latency.
* **Privacy-First:** Your data stays local inside an embedded SQLite database.

---

## Quick Start

git clone 

exports

go run main.go

go build needs special flags for c because of  SQLite dependency

Raspberry pi:
CC=aarch64-linux-gnu-gcc CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -o orga

wsl:/ std linux -> should take the enviroment compiler
CGO_ENABLED=1 GOOS=linux  go build -o orga 


On the PI:
Disclaimer: you need basic network understanding of ports and ips to do this.
troubleshoot:
sudo ss -tulpn | grep :80
sudo setcap 'cap_net_bind_service=+ep' /path/to/your/binary   

Running it on a pi:
export PASSWORD_PEPPER=<own-passwd> 
nohup ./pi_orga -port 80 > orga.log 2>&1 &

test with curl or browser on pi ip:
ip addr gets that
you can define port and should create local cert and allow it in your browsers (for https)
otherwise always go http://<ip>:<port>

### Prerequisites

* [Go 1.20+](https://go.dev/doc/install) installed (only required if building/running from source).

### Environment Variables

Configure your app before launching using standard environment variables:

```bash
# Set logging verbosity (e.g., debug, info, warn, error)
export LOG_LEVEL=debug

# Set your secret pepper string used for password hashing
export PASSWORD_PEPPER="your-super-secret-pepper-string"

known issue:
first login for admin, end in blank page. sorry. will fix sometime
