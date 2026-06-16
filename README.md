# grin-seeder
Dynamic Grin DNS seed server written in Go.

## Installation
```
git clone https://github.com/aeeravsar/grin-seeder
cd grin-seeder
go build -o grin-seeder
sudo cp grin-seeder /usr/local/bin
```
## Usage
Generate `seed.toml` config file:
```
grin-seeder generate-config
```
Edit the settings:
```
[dns]
host        = "127.0.0.1"
port        = 5301
origin      = "seed.example.com"
ns          = "ns1.example.com"
email       = "hostmaster.example.com"
max_records = 24

[node]
# mode = "dynamic" requires `url` and `secret` to receive the online peers list from the node for every `interval`,
# you can comment out `url` and `secret` if you just want to serve a static list of `peers`,
# alive_only = true serves only peers reachable on p2p_port, including hardcoded peers,
# if it is set to false it will serve `peers` from the static list without checking reachability.
mode        = "dynamic" # or "static"
peers       = ["1.2.3.4"]
alive_only  = true # or false
url         = "http://127.0.0.1:3413"
secret      = "/home/user/.grin/main/.api_secret"
interval    = 60
p2p_port    = 3414
check_timeout = 3
min_user_agent = "MW/Grin 5.4.0"
```
Run grin-seeder (sudo for port 53):
```
grin-seeder 
2026/04/19 12:11:34 dns: listening on 127.0.0.1:5301 (UDP+TCP)
2026/04/19 12:11:34 monitor: updated peer list — 27 alive peers
```
