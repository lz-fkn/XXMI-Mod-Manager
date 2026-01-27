# XXMI Manager
*rather simple mod manager for XXMI mods (ZZZ, GI, HSR, WuWa (i think))*

---

made it when i've realised that i need some easy way to toggle on and off the mods on the fly, so here it is. it works by creating a symlink to the mod, simple as that.<br>

used stuff:
* [Golang](https://go.dev/) (go1.25.5 windows/amd64)
* [Wails](https://wails.io/) (v2.11.0)
* [Marked.js](https://github.com/markedjs/marked) (v17.0.1)

---
### build
```
git clone https://github.com/lz-fkn/XXMI-Mod-Manager.git
cd XXMI-Mod-Manager
go install github.com/wailsapp/wails/v2/cmd/wails@latest
go mod tidy
wails build
```