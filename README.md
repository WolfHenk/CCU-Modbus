# CCU-Modbus

CCU-Modbus ist eine Zusatzsoftware für CCU/OpenCCU, mit der Modbus-TCP-Geräte eingebunden und deren Werte in der CCU nutzbar gemacht werden können.

Die Konfiguration erfolgt vollständig über die WebUI der CCU. Es ist keine Cloud, keine externe Datenbank und kein zusätzlicher Server erforderlich.

## Installation

1. Das aktuelle Installationspaket über den folgenden Direktlink herunterladen: [ccu-modbus-latest.tar.gz](https://raw.githubusercontent.com/WolfHenk/CCU-Modbus/be5c544245f63a6cf7656cb999e555fb4efeb1bf/releases/ccu-modbus-latest.tar.gz). Im Repository wird nur das jeweils aktuelle Installationspaket vorgehalten.
2. In der CCU unter **Einstellungen → Systemsteuerung → Zusatzsoftware** die TAR.GZ-Datei auswählen und installieren.
3. Bei der ersten Installation ist ein Neustart der CCU erforderlich, damit die XML-RPC-Schnittstelle von ReGaHss übernommen wird.
4. Anschließend unter **Einstellungen → Systemsteuerung → Zusatzsoftware → CCU-Modbus → Einstellen** die Konfiguration öffnen.

Updates werden auf dem gleichen Weg über den oben verlinkten, commit-festen Download installiert. Dadurch kann kein zwischengespeichertes älteres `latest`-Paket ausgeliefert werden. Die bestehende Konfiguration bleibt dabei erhalten.

## Erstes Modbus-Gerät einrichten

In CCU-Modbus auf **+ Gerät hinzufügen** klicken.

Benötigt werden:

- **Name** – frei wählbarer Gerätename, z. B. `BSM-1216E`
- **IP-Adresse / Host** – Adresse des Modbus-TCP-Geräts
- **Port** – normalerweise `502`
- **Unit-ID** – Modbus-Geräteadresse

Unter **Erweitert** können Timeout, Anzahl der Wiederholungsversuche und der Gerätestatus eingestellt werden.

Mit **Verbindung testen** kann geprüft werden, ob das Gerät erreichbar ist.

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
  - FLOAT32
- **Einheit**
- **Faktor**
- **Offset**
- **Pollingintervall**
- **Register aktiv** – legt fest, ob der Datenpunkt verwendet und abgefragt wird

**Discrete Input** wird in der CCU als neutraler Homematic-Kanal vom Typ `DIGITAL_INPUT` mit einem booleschen `STATE` abgebildet. Dadurch erscheint ein allgemeiner Modbus-Eingang nicht mehr als Tür-/Fensterkontakt. `FALSE` entspricht **inaktiv**, `TRUE` entspricht **aktiv**.

Unter **Erweitert** stehen zusätzlich Byte-Swap und Word-Swap zur Verfügung.

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

Bei 32-Bit-Datentypen wird die Modbus-Adresse automatisch in Zweierschritten weitergezählt.

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

Weitere Registertypen werden bereits von CCU-Modbus gelesen und angezeigt; ihre Abbildung auf zusätzliche CCU-Kanaltypen wird schrittweise erweitert.

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
