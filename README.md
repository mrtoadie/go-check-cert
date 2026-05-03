![assets/header.png](assets/header.png)

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=flat&logo=go&logoColor=white) ![GitHub License](https://img.shields.io/github/license/mrtoadie/go-check-cert) ![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/mrtoadie/go-check-cert/total) ![GitHub Release](https://img.shields.io/github/v/release/mrtoadie/go-check-cert)

## Features
- checks the validity of website certificates
- either a single URL or a (batch) list of URLs from a file
- checks local certificate files
- saves the certificate metadata as a JSON file

## Install
### Build from source
```bash
git clone https://github.com/mrtoadie/go-check-cert.git
cd go-check-cert
go build -o cert-checker
```
### Run directly (without build)
```bash
go run .
```
### Arch Linux
Install from [AUR](https://aur.archlinux.org/packages/cert-checker)
```bash
yay -S cert-checker
```

## Command Line Options
| Flag | Description |
| -f	-file	| Path to a local .pem, .crt, .cer, or .key file |
| -c	-cron |	Interactive Cron setup |
| -list	-ls |	List and manage Cron jobs |
| -ci	-ci-mode | CI/CD Mode (non-interactive, uses urls.txt) |
|-h	-help |	Display help message |

## Examples
Check a single certificate:
```bash
./cert-checker -f ./server.crt
```

Set up cron job(s):
```bash
./cert-checker -c
```
List active Cron jobs:
```bash
./cert-checker -ls
```
## Configuration
The tool automatically creates a configuration file on the first run.

Location: `~/.config/cert-checker/urls.txt`

Format: One URL per line. Comments starting with # are supported.


## Usage
The input is interactive and automatically detects the correct format.

- Press **Enter** to use the default list of URLs (~/.config/cert-checker/urls.txt)

- To check individual URLs: *github.com* or *github.com*, *ubuntu.com*, *example.org*

- Check local certificate file: '~/github.pem'

- Certificate files can also be checked using a flag: cert-checker --file ~/github.pem

Certificate files with the following extensions work: .pem, .cer, .crt, .key

## Output
```bash
=== RESULTS ===

1. forum.linuxguides.de
   Days:  44 | Valid: 26.02.26 → 27.05.26
   Issuer: R12
------------------------------------
2. github.com
   Days:  52 | Valid: 06.03.26 → 03.06.26
   Issuer: Sectigo Public Server Authentication CA DV E36
------------------------------------
=== SUMMARY ===
OK: 0 | Warn: 2 | Exp: 0 | Err: 0
```

## Tested on
:white_check_mark: [Arch Linux](https://archlinux.org/)

:white_check_mark: [Solus](https://getsol.us/)

## License
go-check-cert is licensed under the [MIT License](LICENSE).
