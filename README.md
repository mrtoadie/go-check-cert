![assets/header.png](assets/header.png)

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=flat&logo=go&logoColor=white) ![GitHub License](https://img.shields.io/github/license/mrtoadie/go-check-cert) ![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/mrtoadie/go-check-cert/total) ![GitHub Release](https://img.shields.io/github/v/release/mrtoadie/go-check-cert)

## Features
- checks the validity of website certificates
- either a single URL or a (batch) list of URLs from a file

## Install & Usage
### Arch Linux
Install from [AUR](https://aur.archlinux.org/packages/cert-checker)
```bash
yay -S cert-checker
```
Run
```bash
cert-checker
```
You can enter individual URLs, such as
*github.com* or *github.com, codeberg.org*

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
