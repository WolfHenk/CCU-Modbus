# CCU-Modbus

CCU/OpenCCU-Add-on für ein einfaches, robustes Modbus-Interface.

## Download

Aktuelle Testversion:

**[ccu-modbus-0.1.0-test4.tar.gz](releases/ccu-modbus-0.1.0-test4.tar.gz)**

Die installierbare TAR.GZ-Datei wird nach einem erfolgreichen Build auf `main` direkt im Repository unter `releases/` abgelegt.

## Zusatzsoftware-Darstellung

CCU-Modbus nutzt die normale CCU/OpenCCU-Zusatzsoftware-Mechanik:

- Modbus-Logo und Projektinformationen rechts
- installierte Version
- verfügbare Version
- **Herunterladen**
- **Neustart**
- **Deinstallieren**
- **Einstellen**

Die Schaltflächen entstehen über die Standardfelder des `rc.d ... info`-Aufrufs:

- `Version:`
- `Info:`
- `Update:`
- `Operations:`
- `Config-Url:`

Hinweis: Solange dieses Repository privat ist, kann die CCU GitHub ohne Zugangsdaten nicht selbst nach der neuesten Version fragen. Dann zeigt der automatische Versionscheck `n/a`. Der Mechanismus ist bereits vollständig eingebaut und funktioniert automatisch, sobald die Downloadquelle öffentlich erreichbar ist.

## Stand 0.1.0-test4

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
