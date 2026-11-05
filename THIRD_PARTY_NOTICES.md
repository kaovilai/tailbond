# Third-party notices for adapted source

tailbond is Apache-2.0 (see LICENSE and NOTICE). The files below adapt code
from projects under the BSD 3-Clause License; their notices follow, as that
license requires.

## Untether (ImOstrovskiy/untether)

https://github.com/ImOstrovskiy/untether: a Kotlin Android app and a macOS
menu bar app that start a phone's Wi-Fi hotspot over Bluetooth LE through
Shizuku, with no root. tailbond's "One click from the Mac"
([docs/phone-hub.md](docs/phone-hub.md#one-click-from-the-mac)) adapts:

- `android/app/src/main/kotlin/dev/tailbond/app/ble/HotspotShell.kt` and
  `android/app/src/main/aidl/dev/tailbond/app/ble/IHotspotShell.aidl`: the
  Shizuku user service that starts and stops tethering as the shell user
  (from untether's `HotspotShell.kt` and `IHotspotShell.aidl`; only start,
  stop, state and the network's name and password are kept);
- `android/app/src/main/kotlin/dev/tailbond/app/ble/ShizukuHotspot.kt`: binding
  that service (from untether's `ShizukuHotspotController.kt`).

The Bluetooth protocol, the GATT server and the Mac side are tailbond's own.

```
BSD 3-Clause License

Copyright (c) 2026 Oleksii Ostrovskyi

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its
   contributors may be used to endorse or promote products derived from
   this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

## delta (supershadoe/delta)

https://github.com/supershadoe/delta: untether's hotspot controller follows
delta's approach (Shizuku-wrapped system binders with the `com.android.shell`
caller package), and so does the adapted `HotspotShell.kt`.

```
Copyright 2024, 2025 supershadoe

Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```
