# CCU-Modbus

CCU/OpenCCU-Add-on für ein einfaches, robustes Modbus-Interface.

## Download

Aktuelle Testversion:

**[ccu-modbus-0.1.0-test3.tar.gz](releases/ccu-modbus-0.1.0-test3.tar.gz)**

Die installierbare TAR.GZ-Datei wird nach einem erfolgreichen Build auf `main` direkt im Repository unter `releases/` abgelegt.

## Stand 0.1.0-test3

### WebUI

- Geräteübersicht mit Ampelstatus
- Gerät hinzufügen, bearbeiten und löschen
- Verbindungstest
- Register hinzufügen, bearbeiten und löschen
- Registertest mit Rohwert und berechnetem Wert
- Faktor, Offset, Einheit und Pollingintervall
- Byte-/Word-Swap unter „Erweitert“
- atomisches Speichern mit Sicherung der letzten Konfiguration
- WebUI bleibt beim Neustart des Daemons verfügbar
- Modbus-Logo in Add-on-Liste und Einstellseite

### Modbus-Kern

- Modbus TCP FC01, FC02, FC03 und FC04
- BOOL, UINT16, INT16, UINT32, INT32 und FLOAT32
- harte TCP-Timeouts und begrenzte Retries
- eigener Worker und persistente TCP-Verbindung pro Gerät
- automatische Wiederverbindung
- individuelle Pollingintervalle pro Register
- Fehler einzelner Register bleiben lokal
- lokaler Cache und Status-API

### Noch nicht vorhanden

- CCU XML-RPC / virtuelle Geräte
- Schreibfunktionen
- optimierte Blockabfragen
- Online-Profilbibliothek
- Modbus RTU/RS485

## Architekturgrundsätze

1. Kein externer Dienst ist für den Betrieb erforderlich.
2. Kein CCU/RPC-Aufruf darf auf eine Modbus-Antwort warten.
3. Ein Modbus-Gerät darf kein anderes Gerät blockieren.
4. Ein fehlerhaftes Register darf den Daemon nicht beenden.
5. Gültige Geräte laufen trotz fehlerhafter Konfiguration anderer Geräte weiter.
6. Schreibzugriffe sind standardmäßig deaktiviert.
7. Persistente Daten liegen unter `/usr/local/etc/config/addons/ccu-modbus`.
