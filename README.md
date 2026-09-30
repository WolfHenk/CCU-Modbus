# CCU-Modbus

CCU/OpenCCU-Add-on fuer ein einfaches, robustes Modbus-Interface.

## Download

Aktuelle Testversion:

**[ccu-modbus-0.1.0-test2.tar.gz](releases/ccu-modbus-0.1.0-test2.tar.gz)**

Die installierbare TAR.GZ-Datei wird nach einem erfolgreichen Build auf `main` direkt im Repository unter `releases/` abgelegt.

## Stand

Aktueller Entwicklungsstand: **0.1.0-test2**

### Bereits vorhanden

- eigener Go-Daemon ohne externe Laufzeitabhaengigkeiten
- Modbus TCP FC01, FC02, FC03 und FC04
- BOOL, UINT16, INT16, UINT32, INT32 und FLOAT32
- Faktor und Offset
- Byte- und Word-Swap
- harte TCP-Timeouts und begrenzte Retries
- pro Modbus-Geraet ein eigener Worker
- persistente TCP-Verbindung pro Geraet mit automatischem Reconnect
- individuelle Pollingintervalle pro Register
- Fehler einzelner Register werden lokal behandelt
- lokaler Status-Endpunkt `/status`
- CCU-Paketgeruest mit `update_script` und `rc.d`

### Noch nicht vorhanden

- CCU XML-RPC / virtuelle Geraete
- komfortabler WebUI-Konfigurator
- Schreibfunktionen
- optimierte Blockabfragen
- Online-Profilbibliothek
- Modbus RTU/RS485

## Architekturgrundsaetze

1. Kein externer Dienst ist fuer den Betrieb erforderlich.
2. Kein CCU/RPC-Aufruf darf auf eine Modbus-Antwort warten.
3. Ein Modbus-Geraet darf kein anderes Geraet blockieren.
4. Ein fehlerhaftes Register darf den Daemon nicht beenden.
5. Gueltige Geraete laufen trotz fehlerhafter Konfiguration anderer Geraete weiter.
6. Schreibzugriffe sind standardmaessig deaktiviert.
7. Persistente Daten liegen unter `/usr/local/etc/config/addons/ccu-modbus`.

## Lokal testen

```sh
cp examples/config.json /tmp/ccu-modbus.json
go run ./cmd/ccu-modbusd -config /tmp/ccu-modbus.json
curl http://127.0.0.1:18701/status
```

Nur Konfiguration pruefen:

```sh
go run ./cmd/ccu-modbusd -config examples/config.json -check
```

## Paket bauen

```sh
make package
```

Erzeugt das CCU-Installationsarchiv fuer amd64, arm64 und armv7.
