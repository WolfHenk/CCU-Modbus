# CCU-Modbus

CCU/OpenCCU-Add-on fuer ein einfaches, robustes Modbus-Interface.

## Stand

Aktueller Entwicklungsstand: **0.1.0-test1**

Der erste Teststand konzentriert sich bewusst auf einen kleinen, stabilen Modbus-TCP-Kern.

### Bereits vorhanden

- eigener Go-Daemon ohne externe Laufzeitabhaengigkeiten
- Modbus TCP FC01, FC02, FC03 und FC04
- BOOL, UINT16, INT16, UINT32, INT32 und FLOAT32
- Faktor und Offset
- Byte- und Word-Swap
- harte TCP-Timeouts und begrenzte Retries
- pro Modbus-Geraet ein eigener Worker
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
# Host/Register anpassen und enabled=true setzen
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

Erzeugt ein CCU-Installationsarchiv fuer amd64, arm64 und armv7.
