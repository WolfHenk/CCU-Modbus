# CCU-Modbus

CCU/OpenCCU-Add-on für ein einfaches, robustes Modbus-Interface.

## Download

Aktuelle Testversion:

**[ccu-modbus-0.1.0-test5.tar.gz](releases/ccu-modbus-0.1.0-test5.tar.gz)**

Die installierbare TAR.GZ-Datei wird nach einem erfolgreichen Build auf `main` direkt im Repository unter `releases/` abgelegt.

## Stand 0.1.0-test5

Korrektur gegenüber test4:
- URL-Decodierung für CCU-Session-IDs im CGI ergänzt
- damit funktionieren „Verbindung testen“, „Wert testen“ und Speichern auch bei als `%40...%40` übertragenen Session-IDs

### Zusatzsoftware-Darstellung

CCU-Modbus nutzt die normale CCU/OpenCCU-Zusatzsoftware-Mechanik:

- Modbus-Logo und Projektinformationen rechts
- installierte Version
- verfügbare Version
- **Herunterladen**
- **Neustart**
- **Deinstallieren**
- **Einstellen**

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

### Modbus-Kern

- Modbus TCP FC01, FC02, FC03 und FC04
- BOOL, UINT16, INT16, UINT32, INT32 und FLOAT32
- harte TCP-Timeouts und begrenzte Retries
- eigener Worker und persistente TCP-Verbindung pro Gerät
- automatische Wiederverbindung
- individuelle Pollingintervalle pro Register
- Fehler einzelner Register bleiben lokal
- lokaler Cache und Status-API


## Logo

Für die Darstellung wird das vom Projekt gewünschte Modbus-Logo verwendet: https://seeklogo.com/vector-logo/355093/modbus
