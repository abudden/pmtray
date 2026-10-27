# PasswordMaker Tray

A [PasswordMaker](https://passwordmaker.org/Introduction) password generator
that lives in the notification area (system tray) instead of the browser.
It generates the same passwords as the
[PasswordMaker Pro Chrome extension](https://github.com/passwordmaker/chrome-passwordmaker)
and uses the extension's RDF profile file directly as its configuration.

Windows is the main target; it also runs on Linux (and should build on macOS).

## Using it

Click the tray icon to show the popup:

| Field              | Notes |
|--------------------|-------|
| Password           | Your master password. |
| Verification Code  | 3-letter code to check the master password was typed correctly (can be turned off). |
| Full URL           | Paste the address from the browser's address bar here. |
| Used text          | Worked out from the full URL using the selected profile's URL settings. You can type it instead, which clears the full URL. |
| Use Profile        | Chosen automatically from each profile's *Use With Sites* patterns whenever the URL or used text changes. You can still pick one yourself. |
| Username           | Hidden unless *Show Username Field in Popup* is turned on in the options. |
| Generated Password | Read-only. Press **Ctrl+C** or **Enter** in it, or click **Copy**, to copy the password and close the popup. |

When the popup opens, the cursor is in the **Password** field if no master
password has been entered yet. Otherwise it is in the **Full URL** field with
its text selected, so the usual sequence is:

1. Copy the address in the browser (Ctrl+L, Ctrl+C).
2. Click the tray icon.
3. **Ctrl+V**: the URL is replaced and the cursor jumps to the generated password.
4. **Ctrl+C**: the password is copied and the popup closes.

The tab order is Password → Full URL → Used text → Profile → Generated Password.
**Esc** closes the popup. The full URL, used text and selected profile are kept
while the program is running, so you can reopen the popup, choose another
profile and copy the new password without entering the URL again.

Right-click the tray icon for **Options…** and **Quit**. Starting the program a
second time shows the popup of the copy that is already running.

## Profiles file

Profiles are kept in an RDF file in the same format as the Chrome extension's
*Export Profiles* feature (Firefox PasswordMaker exports can be read too):

* By default the file is `%APPDATA%\pmtray\profiles.rdf` on Windows
  (`~/.config/pmtray/profiles.rdf` on Linux). It's created with the
  extension's two default profiles the first time the program runs.
* *Options → General Options → Use Existing File…* switches to an existing
  file, e.g. one exported from the Chrome extension or kept in a synced folder.
  *Create New File…* saves the current profiles to a new file and uses that.
* The file is checked every second (and whenever the popup opens), so changes
  made by other programs appear straight away. If the file can't be read, the
  previous profiles stay in use and the error is shown in the options window.
* Pressing **Save** (or Add, Remove, Up, Down) on the *Profiles* tab writes the
  file. It is written in the Chrome export format, so it can still be imported
  into the extension. Anything in the file that PasswordMaker Tray doesn't use
  (e.g. Firefox groups or global settings) is not kept when it's saved.

Global settings are stored separately in `settings.json` in the same folder.

## Options

The general options match the Chrome extension, plus a few for the tray version:

* **Set Popup Zoom Level** – scales the popup (and options window).
* **Save Password** – *Per Session* (remember the master password until the
  program exits), *Per Session, Expire* (forget it after a number of minutes),
  *On Disk* (stored in `settings.json`, encrypted with Windows DPAPI for your
  user account; on other systems encrypted with a key in a file next to it),
  *Never Store* (forget it whenever the popup closes).
* **Sort Profiles By** – order of the profile drop-down (the first profile always stays first).
* **Visually Hide Generated Password in Popup**
* **Manually Verify Password Using 3 Letter Code** – when off, a confirmation field is shown instead.
* **Show Password Strength Meter in Popup**
* **Show the Most Recent Profile Modification Time**
* **Keep Master Password Hash** – stores a salted PBKDF2 hash so a mistyped
  master password is reported as a mismatch.
* **Show Username Field in Popup** (new; off by default)
* **Start PasswordMaker When I Log In** (Windows and Linux)

The extension's *Sync Profiles Between Computers* option isn't provided: keep
the profiles file in a synced folder (OneDrive, Dropbox, …) instead.

## Building

PasswordMaker Tray is written in Go with the [Fyne](https://fyne.io) toolkit,
which builds to a single self-contained executable – nothing to install. The
program stays running with its window already created, so the popup appears
immediately when the icon is clicked.

Requirements: Go (see `go.mod`) and a C compiler (Fyne uses OpenGL through cgo).

**Windows** (with a MinGW-w64 gcc, e.g. from [MSYS2](https://www.msys2.org/) or
[WinLibs](https://winlibs.com/), on the `PATH`):

```
go build -trimpath -ldflags "-s -w -H=windowsgui" -o pmtray.exe .
```

The `rsrc_windows_*.syso` files add the icon and manifest to the executable.
They're generated from `winres/winres.json` with
[go-winres](https://github.com/tc-hib/go-winres) (`go-winres make --arch amd64,arm64`).

**Linux** (Debian/Ubuntu packages shown):

```
sudo apt install gcc libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev
go build -trimpath -ldflags "-s -w" -o pmtray .
```

Cross-compiling a Windows executable from Linux also works:

```
sudo apt install gcc-mingw-w64-x86-64
CGO_ENABLED=1 GOOS=windows CC=x86_64-w64-mingw32-gcc go build -trimpath -ldflags "-s -w -H=windowsgui" -o pmtray.exe .
```

The GitHub Actions workflow builds Windows and Linux executables for every
push, and attaches them to a release when a `v*` tag is pushed.

Run the tests with `go test ./...`. The password generation tests use vectors
from a literal port of the extension's JavaScript, including the quirks of its
"HMAC-SHA-256 Version 1.5.1" and "MD5 Version 0.6" algorithms.

The tray icon is drawn by `go run ./tools/mkicon internal/ui/assets/icon.png`.

## Layout

| Path              | Contents |
|-------------------|----------|
| `internal/pwm`    | The PasswordMaker algorithm: hashes, l33t, URL handling, site patterns, strength meter. |
| `internal/rdf`    | Reading and writing RDF profile files. |
| `internal/config` | Global settings, the profiles store (watches the RDF file), start-at-login. |
| `internal/ui`     | Tray icon, popup and options window. |
| `internal/single` | Single-instance handling. |

## Credits

The password algorithm, l33t tables and top-level domain list are ported from
the PasswordMaker Pro Chrome extension, which is licensed under the LGPL 3.0.
