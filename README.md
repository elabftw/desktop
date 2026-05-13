# eLabFTW Desktop

> ⚠️ This project is experimental and not the primary way to run eLabFTW.
> The recommended production setup is the Docker-based web application. See [official documentation](https://doc.elabftw.net/).

## Development Setup

---

## Prerequisites

### Go

**Minimum version:** Go **1.22+**

Check:

```bash
go version
```

---

### Node.js

- **Minimum version:** Node.js **18+**
- Recommended: Node.js 20+

```bash
node -v
npm -v
```

---

### Wails CLI

Install:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Ensure it's in your PATH:

```bash
export PATH="$PATH:$HOME/go/bin"
```

Verify:

```bash
wails doctor
```

---

## Linux dependencies

Wails requires system libraries (GTK + WebKitGTK).

### Fedora (≥ 40)

```bash
sudo dnf install gtk3-devel webkit2gtk4.1-devel pkgconf-pkg-config gcc gcc-c++
```

### Ubuntu / Debian

```bash
sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev pkg-config
```

### Archlinux

```bash
sudo pacman -Syu webkit2gtk-4.1
```

---

## Setup

```bash
git clone https://github.com/elabftw/desktop.git
cd desktop
#install frontend dependencies
cd frontend
npm install
# go back to parent directory
cd ..
```

---

## Run the application

### Standard (most Linux distros)

```bash
wails dev
```

---

### ⚠️ Fedora users (important)

On Fedora ≥ 40, WebKitGTK **4.1** is used instead of **4.0**.
Wails defaults to 4.0, so you must run:

```bash
wails dev -tags webkit2_41
```

If you don't use this flag, you may see errors like:

```text
Package 'webkit2gtk-4.0' not found
Build error - exit status 1
```

---

## 🖥 What to expect

- A **native desktop window** should open automatically
- The app is **not meant to run in a browser**

---

## 🌐 Dev URLs

During development, Wails exposes:

- Frontend only (no Go backend):
  http://localhost:5173/

- Internal dev server (Wails):
  http://localhost:34115

⚠️ These are for debugging only.
The actual app runs in the native window.
