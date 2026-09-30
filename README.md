# CCU-Modbus

CCU/OpenCCU-Add-on für ein einfaches, robustes Modbus-Interface.

## Download

Aktuelle Testversion:

**[ccu-modbus-0.1.1.tar.gz](releases/ccu-modbus-0.1.1.tar.gz)**

Die installierbare TAR.GZ-Datei wird nach einem erfolgreichen Build auf `main` direkt im Repository unter `releases/` abgelegt.

## Stand 0.1.1

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


## Direkter Download

Der CCU-Button **Herunterladen** verweist direkt auf `releases/ccu-modbus-latest.tar.gz`. Diese Datei wird bei jedem erfolgreichen Build aktualisiert und soll vom Browser unmittelbar als TAR.GZ heruntergeladen werden.


## Virtuelle CCU-Geräte

Ab test10 stellt CCU-Modbus konfigurierte Modbus-Geräte über eine eigene XML-RPC-Schnittstelle als virtuelle CCU-Geräte bereit.

- Gerätetyp: `CCU-MODBUS-...`
- Kanal 0: Wartungs-/Statuskanal
- konfigurierte BOOL-Coils: CCU-Schaltkanäle mit `STATE`
- CCU-Schaltbefehle werden nur in die Geräte-Worker-Warteschlange gelegt; der CCU-RPC-Aufruf wartet niemals auf Modbus
- Rückmeldungen kommen aus dem lokalen Polling-Cache
- erste Registrierung der Schnittstelle erfordert einmalig einen Neustart, damit ReGaHss die neue `InterfacesList.xml` einliest


## Lizenz

CCU-Modbus steht unter der **PolyForm Noncommercial License 1.0.0**.

- Copyright (c) 2026 Wolfram Henkel
- private und sonstige nichtkommerzielle Nutzung erlaubt
- Änderungen und Weitergabe erlaubt
- Copyright- und Lizenzhinweis müssen erhalten bleiben
- **kommerzielle Nutzung ist nicht gestattet**

Der Name Modbus und das Modbus-Logo sind davon ausgenommen und bleiben Fremdmarken/-assets; siehe `THIRD_PARTY_NOTICES`.


## Lizenz

CCU-Modbus steht unter der **CCU-Modbus Non-Commercial License 1.0**.
Private, gemeinnützige und sonstige nichtkommerzielle Nutzung ist gestattet.
Kommerzielle Nutzung ist nur mit vorheriger schriftlicher Genehmigung des
Urheberrechtsinhabers zulässig.

Das Modbus-Logo ist ein fremdes Marken-/Logo-Asset und wird durch die
CCU-Modbus-Lizenz nicht mitlizenziert. Siehe `THIRD_PARTY_NOTICES`.
