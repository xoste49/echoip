# echoip

[![ci](https://github.com/xoste49/echoip/actions/workflows/ci.yml/badge.svg)](https://github.com/xoste49/echoip/actions/workflows/ci.yml)

A simple service for looking up your IP address. This is the code that powers
https://ifconfig.co.

## Usage

Just the business, please:

```
$ curl ifconfig.co
127.0.0.1

$ http ifconfig.co
127.0.0.1

$ wget -qO- ifconfig.co
127.0.0.1

$ fetch -qo- https://ifconfig.co
127.0.0.1

$ bat -print=b ifconfig.co/ip
127.0.0.1
```

Country and city lookup:

```
$ curl ifconfig.co/country
Elbonia

$ curl ifconfig.co/country-iso
EB

$ curl ifconfig.co/city
Bornyasherk

$ curl ifconfig.co/asn
AS31337

$ curl ifconfig.co/asn-org
Dilbert Technologies
```

As JSON:

```
$ curl -H 'Accept: application/json' ifconfig.co  # or curl ifconfig.co/json
{
  "city": "Bornyasherk",
  "country": "Elbonia",
  "country_iso": "EB",
  "ip": "127.0.0.1",
  "ip_decimal": 2130706433,
  "asn": "AS31337",
  "asn_org": "Dilbert Technologies"
}
```

Port testing:

```
$ curl ifconfig.co/port/80
{
  "ip": "127.0.0.1",
  "port": 80,
  "reachable": false
}
```

Speed test:

```
$ curl -o /dev/null ifconfig.co/10mb
$ curl -o /dev/null ifconfig.co/100mb
$ curl -o /dev/null ifconfig.co/500mb
$ curl -o /dev/null ifconfig.co/1gb
```

Any size is supported — use a number followed by `mb` or `gb` (e.g. `50mb`, `150mb`, `2gb`). Maximum is 10 GB.

Connectivity check (returns `204 No Content`, similar to `google.com/generate_204`):

```
$ curl -o /dev/null -w "%{http_code}" ifconfig.co/generate_204
204
```

Pass the appropriate flag (usually `-4` and `-6`) to your client to switch
between IPv4 and IPv6 lookup.

## Features

* Easy to remember domain name
* Fast
* Supports IPv6
* Supports HTTPS
* Supports common command-line clients (e.g. `curl`, `httpie`, `ht`, `wget` and `fetch`)
* JSON output
* ASN, country and city lookup, using data from MaxMind
* Port testing
* Speed test downloads (any size in MB or GB, up to 10 GB)
* Connectivity check endpoint (`/generate_204`)
* All endpoints (except `/port`) can return information about a custom IP address specified via `?ip=` query parameter
* Open source under the [BSD 3-Clause license](https://opensource.org/licenses/BSD-3-Clause)

## Why?

* To scratch an itch
* An excuse to use Go for something
* Faster than ifconfig.me and has IPv6 support

## Building

Compiling requires the [Golang compiler](https://golang.org/) to be installed.
This package can be installed with:

`go install github.com/xoste49/echoip/...@latest`

For more information on building a Go project, see the [official Go
documentation](https://golang.org/doc/code.html).

## Docker image

A Docker image is available on [Docker
Hub](https://hub.docker.com/r/xoste49/echoip), which can be downloaded with:

`docker pull xoste49/echoip:latest`

## Docker Run

```shell
git clone https://github.com/xoste49/echoip
cd echoip
make geoip-download
docker run -d --name echoip --restart unless-stopped -p 8080:8080 -v $(pwd)/data:/opt/echoip/data xoste49/echoip:latest -c data/city.mmdb -f data/country.mmdb -a data/asn.mmdb
```

## Run without Docker

```shell
make geoip-download
make run
```

## Geolocation data

`echoip` uses the MaxMind GeoIP databases to show additional information about
IP addresses, such as registered country/city and ASN details.

The databases can be downloaded with:

`make geoip-download`

The databases are downloaded from the https://github.com/P3TERX/GeoLite.mmdb repository.

### Usage

```
$ echoip -h
Usage of echoip:
  -C int
        Size of response cache. Set to 0 to disable
  -H value
        Header to trust for remote IP, if present (e.g. X-Real-IP)
  -P    Enables profiling handlers
  -a string
        Path to GeoIP ASN database
  -c string
        Path to GeoIP city database
  -f string
        Path to GeoIP country database
  -l string
        Listening address (default ":8080")
  -p    Enable port lookup
  -r    Perform reverse hostname lookups
  -s    Show sponsor logo
  -t string
        Path to template dir (default "html")
```
