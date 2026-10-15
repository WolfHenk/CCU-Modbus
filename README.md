# CCU-Modbus

CCU-Modbus ist eine Zusatzsoftware für CCU/OpenCCU, mit der Modbus-TCP-Geräte eingebunden und deren Werte in der CCU nutzbar gemacht werden können.

Die Konfiguration erfolgt vollständig über die WebUI der CCU. Es ist keine Cloud, keine externe Datenbank und kein zusätzlicher Server erforderlich.

## Aktuelle Version: 0.2.1

[**Release 0.2.1 herunterladen**](https://github.com/WolfHenk/CCU-Modbus/releases/tag/v0.2.1) · [**Installationspaket 0.2.1 (TAR.GZ)**](https://github.com/WolfHenk/CCU-Modbus/releases/download/v0.2.1/ccu-modbus-0.2.1.tar.gz) · [**SHA-256-Prüfsumme**](https://github.com/WolfHenk/CCU-Modbus/releases/download/v0.2.1/ccu-modbus-0.2.1.tar.gz.sha256)

**Neu in 0.2.1:** Ein-Klick-Selbstupdate in der CCU-Modbus-WebUI. Die Funktion fragt veröffentlichte GitHub-Releases ab, vergleicht die Version, lädt nach Bestätigung das Paket herunter, prüft die SHA-256-Summe und übergibt es an den OpenCCU-Installer. Ein eigenständiges Go-Programm übernimmt die Aktualisierung; Python wird auf der CCU nicht benötigt. Der Modbus-Betrieb bleibt ohne Internetverbindung nutzbar.

**Teststand:** Installation des 0.2.1-Entwicklungsstands auf einer separaten ARM64-OpenCCU, Daemon-Betrieb und Release-Abfrage sind bestanden. Der vollständige Ein-Klick-Updatevorgang inklusive Installationsabschluss und Rückfallverhalten ist noch **nicht abschließend geprüft**. Die reguläre CCU-Zusatzsoftwareverwaltung erkennt das veröffentlichte 0.2.1 als neue Version gegenüber einer absichtlich als 0.1.1 bezeichneten Testinstallation.

## 8-Bit-Register (ab 0.1.31)

Fuer UINT8/INT8 liest das Add-on genau ein Modbus-16-Bit-Register. Standardmaessig wird dessen niederwertiges Byte verwendet; mit der bestehenden Option **Byte-Swap** das hoeherwertige Byte. INT8 wird als vorzeichenbehaftete Zweierkomplementzahl interpretiert. Beide Datentypen erscheinen wie 16- und 32-Bit-Werte numerisch in der CCU. Die Modbus-Adressierung bleibt wortbasiert.

## Numerische CCU-Register (ab 0.1.30)

Input- und Holding-Register mit UINT16, INT16, UINT32, INT32 oder FLOAT32 erhalten einen lesbaren numerischen CCU-Datenpunkt `VOLTAGE` (FLOAT). Die Bezeichnung `VOLTAGE` ist durch die verwendete Homematic-Kanalvorlage vorgegeben und bezeichnet keine physikalische Spannung; der Wert bleibt der konfigurierte Modbus-Messwert. UINT64 und INT64 erhalten wegen der begrenzten IEEE-754-FLOAT-Genauigkeit stattdessen einen lesbaren CCU-Datenpunkt `VALUE` (STRING) mit exakter Dezimaldarstellung bei Faktor 1 und Offset 0. Bei skalierten 64-Bit-Werten gilt weiterhin die bereits vorhandene FLOAT-Berechnung im Modbus-Kern. Bestehende binaere Kanaele behalten ihre Kanalnummern. Die praktische CCU-Darstellung ist separat zu pruefen.

## Eingangszustand umkehren

Ab Version **0.1.29** gibt es bei **Discrete Input** im Registerdialog die Option **Eingangszustand in der CCU umkehren (1 ↔ 0)**. Der gelesene Modbus-Rohwert bleibt unveraendert. Nur die boolesche Darstellung in der CCU und in der Werteansicht wird invertiert; Ausgaenge bleiben unberuehrt. Standardmaessig ist die Option ausgeschaltet, vorhandene Register bleiben dadurch unveraendert.

## Geraete loeschen

Ab Version **0.1.28** entfernt das Loeschen eines kompletten Modbus-Geraets in der OpenCCU-Geraeteverwaltung auch den zugehoerigen Geraeteeintrag einschliesslich seiner gespeicherten CCU-Seriennummer aus der persistenten Add-on-Konfiguration. Das Loeschen eines einzelnen Kanals entfernt das Geraet nicht. Bei einem Fehler beim Speichern wird die Loeschung abgelehnt.

## Installation

1. Das [veröffentlichte Installationspaket 0.2.1](https://github.com/WolfHenk/CCU-Modbus/releases/download/v0.2.1/ccu-modbus-0.2.1.tar.gz) herunterladen. Der frühere commit-feste `latest`-Link bleibt über die Versionshistorie nachvollziehbar; für 0.2.1 gilt der veröffentlichte Release als Installationsquelle.
2. In der CCU unter **Einstellungen → Systemsteuerung → Zusatzsoftware** die TAR.GZ-Datei auswählen und installieren.
3. Bei der ersten Installation ist ein Neustart der CCU erforderlich, damit die XML-RPC-Schnittstelle von ReGaHss übernommen wird.
4. Anschließend unter **Einstellungen → Systemsteuerung → Zusatzsoftware → CCU-Modbus → Einstellen** die Konfiguration öffnen.

Bei späteren Updates bleibt die vorhandene Modbus-Konfiguration erhalten. Die CCU-Zusatzsoftwareverwaltung kann veröffentlichte Versionen erkennen und ein Installationspaket zum Herunterladen anbieten.

### Selbstupdate in CCU-Modbus (neu in 0.2.1)

In der CCU-Modbus-WebUI gibt es den Bereich **Software-Update**:

1. **Auf Update prüfen** fragt das neueste veröffentlichte GitHub-Release ab.
2. Ist eine höhere Version verfügbar, wird **Update installieren** aktiv.
3. Nach einer ausdrücklichen Bestätigung lädt der Update-Worker Archiv und Prüfsummendatei herunter, prüft sie und startet den OpenCCU-Installer.
4. Die WebUI zeigt den Update-Status und gegebenenfalls einen erforderlichen Neustart an.

Die Prüfung erfolgt auf Knopfdruck; für die normale Modbus-Kommunikation ist weder GitHub noch eine Cloud erforderlich. **Hinweis:** Der vollständige Updateablauf ist noch in der praktischen End-to-End-Abnahme. Bis dahin bleibt die manuelle Installation über die CCU-Zusatzsoftwareverwaltung der bestätigte Installationsweg.

## Erstes Modbus-Gerät einrichten

In CCU-Modbus auf **+ Gerät hinzufügen** klicken.

Benötigt werden:

- **Name** – frei wählbarer Gerätename, z. B. `BSM-1216E`
- **IP-Adresse / Host** – Adresse des Modbus-TCP-Geräts
- **Port** – normalerweise `502`
- **Unit-ID** – Modbus-Geräteadresse

Unter **Erweitert** können Timeout, Anzahl der Wiederholungsversuche und der Gerätestatus eingestellt werden.

Mit **Verbindung testen** kann geprüft werden, ob das Gerät erreichbar ist.

Neue Modbus-Geräte erhalten ab Version 0.1.27 eine dauerhaft gespeicherte, höchstens 14-stellige CCU-Geräteadresse aus den letzten beiden IP-Oktetten, Port und Unit-ID, z. B. `MB003203504001` für `...3.203`, Port `504`, Unit-ID `1`. Diese Adresse bleibt unverändert, wenn später weitere Units auf derselben IP hinzukommen. Bereits vor 0.1.27 vorhandene, eindeutige Geräte behalten bei der Migration ihre bisherige Seriennummer, damit bestehende ISE-IDs, Programme, Räume und Gewerke nicht unnötig verändert werden.

## Register, Eingänge und Ausgänge anlegen

Innerhalb des Geräts auf **+ Register hinzufügen** klicken.

Für jeden Datenpunkt werden festgelegt:

- **Bezeichnung** – frei wählbarer Name, z. B. `Eingang 1`, `Ausgang 1` oder `Temperatur Vorlauf`
- **Registertyp**
  - Coil
  - Discrete Input
  - Holding Register
  - Input Register
- **Registernummer**
- **Datentyp**
  - BOOL
  - UINT16
  - INT16
  - UINT32
  - INT32
  - UINT64
  - INT64
  - FLOAT32
- **Einheit**
- **Faktor**
- **Offset**
- **Pollingintervall**
- **Register aktiv** – legt fest, ob der Datenpunkt verwendet und abgefragt wird

**Discrete Input** wird in der CCU als neutraler Homematic-Kanal vom Typ `DIGITAL_INPUT` mit einem booleschen `STATE` abgebildet. Dadurch erscheint ein allgemeiner Modbus-Eingang nicht mehr als Tür-/Fensterkontakt. `FALSE` entspricht **inaktiv**, `TRUE` entspricht **aktiv**.

Unter **Erweitert** stehen zusätzlich Byte-Swap, Word-Swap und für 64-Bit-Werte DWord-Swap zur Verfügung. Byte-Swap vertauscht die beiden Bytes jedes 16-Bit-Registers. Word-Swap vertauscht die beiden 16-Bit-Wörter innerhalb jedes 32-Bit-Blocks. DWord-Swap vertauscht bei 64-Bit-Werten die beiden 32-Bit-Hälften. Die Optionen sind kombinierbar und decken damit die üblichen Herstellerreihenfolgen ab.

Mit **Wert testen** kann ein Register unmittelbar gelesen und die Konfiguration geprüft werden.

## Mehrere aufeinanderfolgende Register automatisch anlegen

Bei gleichartigen Ein- oder Ausgängen muss nicht jeder Kanal einzeln eingegeben werden.

Beispiel:

- Bezeichnung: `Eingang 1`
- Typ: `Discrete Input`
- Adresse: `0`
- **Mehrere aufeinanderfolgende Register anlegen**
- Gesamtanzahl: `16`

CCU-Modbus erzeugt daraus automatisch:

```text
Eingang 1   Adresse 0
Eingang 2   Adresse 1
Eingang 3   Adresse 2
...
Eingang 16  Adresse 15
```

Alle übrigen Registereinstellungen werden übernommen. Anschließend kann jeder erzeugte Kanal einzeln angepasst werden.

Bei 32-Bit-Datentypen wird die Modbus-Adresse automatisch in Zweierschritten, bei 64-Bit-Datentypen in Viererschritten weitergezählt.

## Konfiguration speichern

Änderungen werden zunächst nur im Dialog vorgenommen.

Mit **Übernehmen** wird das bearbeitete Gerät in die aktuelle Konfiguration übernommen. Mit **Änderungen speichern** wird die gesamte Konfiguration dauerhaft auf der CCU gespeichert.

Die Konfiguration liegt unter:

```text
/usr/local/etc/config/addons/ccu-modbus/config.json
```

Sie bleibt bei Updates erhalten.

## Darstellung in der CCU

Jedes konfigurierte Modbus-Gerät wird als virtuelles CCU-Gerät der Geräteklasse **ModBus** bereitgestellt.

Das virtuelle Gerät besitzt:

- Kanal 0 als Wartungs-/Statuskanal
- BOOL-Coils als schaltbare CCU-Kanäle
- BOOL-Discrete-Inputs als nur lesbare CCU-Kanäle
- frei vergebene Kanalnamen aus der Modbus-Konfiguration

Schaltbefehle der CCU werden nicht synchron direkt auf Modbus ausgeführt. Sie werden an den jeweiligen Geräte-Worker übergeben. Dadurch wartet ReGaHss nicht auf ein langsames oder ausgefallenes Modbus-Gerät.

Die Rückmeldungen an die CCU stammen aus dem lokalen Polling-Cache.

**Räume und Gewerke werden ausschließlich in der normalen CCU-Weboberfläche zugeordnet.** CCU-Modbus verändert diese Zuordnungen beim Speichern nicht. Auch bei Änderungen der Modbus-Kanalstruktur werden bestehende CCU-Kanäle nicht mehr als komplettes Gerät ab- und neu angemeldet, damit ihre Raum- und Gewerkzuordnungen erhalten bleiben.

Weitere Registertypen werden bereits von CCU-Modbus gelesen und angezeigt; ihre Abbildung auf zusätzliche CCU-Kanaltypen wird schrittweise erweitert. UINT64-/INT64-Rohwerte werden als Dezimaltext ausgegeben, damit große 64-Bit-Ganzzahlen in der WebUI nicht durch die JavaScript-Zahlendarstellung an Genauigkeit verlieren.

## Status und Diagnose

In der Geräteübersicht zeigt eine Ampel den Zustand:

- **Grün** – Gerät erreichbar
- **Gelb** – Gerät erreichbar, einzelne Register mit Fehlern
- **Rot** – Gerät nicht erreichbar oder Konfigurationsfehler

Zu jedem Register werden aktueller Wert und Qualität angezeigt.

Kommunikationsfehler werden lokal behandelt. Ein fehlerhaftes Register oder ein ausgefallenes Modbus-Gerät blockiert keine anderen Geräte.

## Unterstützte Modbus-Funktionen

Aktuell unterstützt:

- FC01 – Read Coils
- FC02 – Read Discrete Inputs
- FC03 – Read Holding Registers
- FC04 – Read Input Registers
- FC05 – Write Single Coil

Modbus TCP ist die derzeit unterstützte Transportart. Modbus RTU/RS485 ist für spätere Erweiterungen vorgesehen.

## Deinstallation

CCU-Modbus kann über **Einstellungen → Systemsteuerung → Zusatzsoftware** deinstalliert werden.

Dabei werden der Dienst, die Zusatzsoftware-Einträge und die von CCU-Modbus bereitgestellten virtuellen Geräte entfernt.

Ein in CCU-Modbus gelöschtes Gerät soll ebenfalls aus der CCU-Geräteliste verschwinden.

## Lizenz

CCU-Modbus steht unter der **CCU-Modbus Non-Commercial License 1.0**.

Private, gemeinnützige und sonstige nichtkommerzielle Nutzung ist erlaubt. Kommerzielle Nutzung ist nur mit vorheriger schriftlicher Genehmigung des Urheberrechtsinhabers zulässig.

Der Name **Modbus** und das Modbus-Logo sind fremde Marken bzw. Assets und werden durch die CCU-Modbus-Lizenz nicht mitlizenziert. Einzelheiten stehen in `LICENSE` und `THIRD_PARTY_NOTICES`.

Copyright © 2026 Wolfram Henkel
### Entwicklung und Releases

Entwicklung, Tests und Paketierung erfolgen lokal; GitHub dient als Versionsverwaltung und Veröffentlichungsplattform. Siehe [AGENTS.md](AGENTS.md).

Releases werden über Tags im Format `vX.Y.Z` gebaut und enthalten ein versionsgebundenes Installationsarchiv mit SHA-256-Prüfsumme. Die CCU prüft veröffentlichte Releases, nicht den wechselnden Entwicklungsstand des Branches `main`.

### Testnachweise

- [Testprotokoll Version 0.2.0 vom 15. Oktober 2026](docs/CCU-Modbus_0.2.0_Testprotokoll_2026-10-15.md): 63 dokumentierte bestandene Prüfpunkte einschließlich Installation des veröffentlichten 0.2.0-Pakets auf ARM64-OpenCCU. Nicht durchgeführte Plattform- und Zusatzprüfungen sind ausgewiesen.
- **Version 0.2.1:** Quellcode- und Buildtests sowie ARM64-Testinstallation und Update-Versionsprüfung bestanden. Der vollständige Ein-Klick-Selbstupdate-Test ist noch offen; aus der reinen Anzeige einer neuen Version wird kein bestandener Installationstest abgeleitet.
