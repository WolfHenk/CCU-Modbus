# CCU-Modbus 0.2.0 — Testprotokoll

**Datum:** 15. Oktober 2026
**Prüfgrundlage:** [homematic-community/ccu-addon-howto, Kapitel 08: Testing](https://github.com/homematic-community/ccu-addon-howto/blob/master/docs/08-testing.md), zusammen mit den Kapiteln über Paketinstallation, RC-Skript, WebUI, Systemintegration und Updates.
**Prüfstand:** `heizungmaster` (Build und Tests), OpenCCU `192.168.7.78` (Pi Zero 2 W, ARM64, OpenCCU 3.89.11.20260919), vierkanaliger Modbus-TCP-Testbaustein `192.168.3.206:502`, Unit 1, Samsung Galaxy Tab 2 mit Firefox/Android 6.0.1.
**Nicht angefasst:** Produktiv-OpenCCU `192.168.3.200`.
**Versionsstatus:** Lokal `VERSION=0.2.0`; `make package` erfolgreich; erzeugte Datei auf `heizungmaster`: `build/ccu-modbus-0.2.0.tar.gz`. Die Installation *dieses neu benannten 0.2.0-Pakets* auf der Test-CCU wurde von der Ausführungsumgebung blockiert und ist **noch nicht bestanden**. Die nachstehenden Hardwaretests betreffen den inhaltlich geprüften vorigen lokalen Test-Build mit Kennung 0.1.32, dessen Korrekturen im neuen 0.2.0-Quellstand enthalten sind. Keine Git-Veröffentlichung erfolgt.

## 1. Quellcode- und statische Prüfungen — bestanden

1. `go test ./...`: alle im Repository vorhandenen Go-Tests bestanden, darunter Konfiguration, Engine und virtuelle CCU-Geräte.
2. `go vet ./...`: ohne Befund.
3. Gezielter Regressionstest `TestActiveUsesExistingConnection`: Wiederverwendung des bestehenden Modbus-Clients für Verbindungstest und Registertest, **ohne zusätzliche TCP-Verbindung**; geänderte Unit-ID wird als separater Test behandelt.
4. Python-Test `packaging/test_refresh_addon_versions.py`: Trigger, Wiederholung, gleichzeitige Auslösung, Fehlerszenarien und Reihenfolge zu Reboot/Installation bestanden.
5. Python-Test `packaging/test_lifecycle.py`: Early Init, Preflight, Update-/Restart-Logik und Releasebezug bestanden.
6. Python-Test `packaging/test_webui_security.py`: Sitzungsschutz, erforderliche POST-Methode und Weitergabe von Backend-HTTP-Fehlercodes bestanden.
7. `node --check packaging/www/app.js`: JavaScript-Syntax bestanden.
8. Automatische Prüfung auf hart kodierte WebUI-Versionsangaben bestanden.
9. ShellCheck mit `-S error -s sh` für `update_script`, `rc.d/ccu-modbus`, `bin/register_devtype`: keine Fehler.
10. `make test` im lokalen Quellstand nach den Korrekturen: Exit-Code **0**.

*Abgrenzung:* Damit sind die vorhandenen automatischen Quellcodeprüfungen bestanden, nicht jede denkbare Last-, Sicherheits- oder Protokollvariante.

## 2. Build und Paket — bestanden

11. Go-Cross-Build für **linux/amd64**: bestanden.
12. Go-Cross-Build für **linux/arm64**: bestanden.
13. Go-Cross-Build für **linux/arm (GOARM=7)**: bestanden.
14. Erzeugung des Pakets **`ccu-modbus-0.2.0.tar.gz`** mit `make package`: Exit-Code **0**.
15. `VERSION` im lokalen Repository lautet **0.2.0**; eingebettete Go-Binärversionsangabe wird mit `-X main.version=0.2.0` gebaut.

*Abgrenzung:* Cross-Build-Erfolg ersetzt keinen Laufzeittest auf physischer x86_64- oder originaler CCU3-Hardware.

## 3. Integration auf echter OpenCCU — bestanden (inhaltlich geprüfter Test-Build)

16. Erstinstallation über `/bin/install_addon`: Rückgabecode **10** (Neustartbedarf), Add-on-Dateien, RC-Skript und WebUI-Verknüpfung angelegt.
17. Neustart nach Erstinstallation: Dienst startet automatisch; Port `127.0.0.1:18701` erreichbar.
18. XML-RPC-Schnittstelle in `InterfacesList.xml` registriert; genau **ein** Eintrag.
19. Realer Vierkanal-Modbus-Baustein eingerichtet, `192.168.3.206:502`, Unit 1; Seriennummer **`MB003206502001`**.
20. Vier Discrete Inputs (Funktion 02) lesbar, Qualität GOOD.
21. Vier Coils (Funktion 01) lesbar, Qualität GOOD.
22. Jeder der vier Ausgänge einzeln über XML-RPC EIN und AUS geschaltet; Rücklesen jeweils korrekt; Endzustand AUS.
23. Gleichzeitige Befehle über das Add-on: **3 Durchläufe × 4 EIN + 4 AUS = 24** erfolgreich ausgeführte Schaltbefehle; begleitende Statusabfragen erfolgreich; Fehlerzähler 0.
24. XML-RPC `listDevices`, `getParamsetDescription`, `getParamset`, `getValue`: erfolgreich; ungültiger Kanal führt zum erwarteten Fehler.
25. Kanaltypen korrekt: DIGITAL_INPUT nur lesbar, SWITCH/STATE schreibbar; Maintenance-Kanal vorhanden.
26. ReGaHSS erkennt Gerät und alle neun Kanäle nach Systemstart.
27. ReGa-Datenpunkt-Ausgang 1 folgt dem XML-RPC-Schaltvorgang: **FALSE → TRUE → FALSE**.
28. Echtes Homematic-Programm auf Test-CCU ausgelöst: Modbus-Ausgangsereignis setzte Test-Systemvariable von FALSE auf TRUE. Testprogramm und Variable anschließend nachweislich entfernt.
29. Nach regulärem Neustart der Test-CCU automatische Wiederverbindung zum realen Baustein: ONLINE, acht Register GOOD, Fehlerzähler 0.

## 4. Multi-Unit, Fehlerfälle und Wiederanlauf — bestanden

30. Zwei Modbus-Units **1 und 2** bei identischer Simulator-IP und identischem Port als zwei unterschiedliche virtuelle CCU-Geräte angelegt.
31. Beide Simulatorgeräte ONLINE; Holding-Registerwert **1234** mit Qualität GOOD.
32. Beim Abschalten des Simulators wechselten beide auf OFFLINE / READ_ERROR.
33. Nach Wiederanlauf des Simulators automatische Rückkehr beider Geräte auf ONLINE / GOOD ohne Add-on-Neustart.
34. Reales Vierkanalgerät blieb während des Simulator-Ausfalls ONLINE.
35. Temporäre Simulatorgeräte entfernt; nachfolgend nur der reale Baustein in Konfiguration und XML-RPC-Liste.
36. Bei einem verbundenen Baustein mit offenbar begrenzter gleichzeitiger TCP-Verbindungszahl funktionieren die **korrigierten** WebUI-Verbindungstests und Registertests über den vorhandenen Client.
37. Bei geänderter Unit-ID verwendet der Test nicht die laufende Verbindung, sondern führt den unabhängigen Verbindungsversuch aus (die Testhardware wies diesen wie erwartet ab).

*Abgrenzung:* Zwei unabhängige, gleichzeitig aktive Modbus-TCP-Clients mit erfolgreich ausgehandelten Verbindungen wurden **nicht** nachgewiesen.

## 5. Update, Lifecycle, Deinstallation — bestanden (inhaltlich geprüfter Test-Build)

38. Installation über bestehende Version ohne Neustartpflicht: Installer-Exit **0**; Dienst danach running.
39. Update von **0.1.31 auf 0.1.32** mit Rückgabecode 0 und laufendem Dienst.
40. Mehrere Updateversuche mit Konfigurationserhalt: SHA-256 der Konfigurationsdatei vorher/nachher identisch.
41. Deinstallation: Installationsverzeichnis, RC-Skript und registrierter Schnittstelleneintrag entfernt.
42. Frische Neuinstallation nach Deinstallation: Exit **10**; Wiederherstellung gesicherter Konfiguration und Dienststart erfolgreich.
43. Neustart nach Neuinstallation: Dienst startet automatisch; ReGa erkennt Gerät, XML-RPC meldet ein Modbus-Gerät plus neun Kanäle.
44. Browser-Speichern einer realen Testkonfiguration: Pollintervall Eingang 1 **5 → 6 s**; Erfolgsmeldung, Dienstneustart, Wert nach Seiten-Reload und über Backend **6 s**.
45. Originalintervall im Browser zurück auf **5 s** gespeichert; alle acht Pollintervalle anschließend wieder **5 s**.
46. Aktualisierungen mit der inhaltlich geprüften Korrektur und nachfolgender Konfigurationserhaltung erfolgreich; 0.2.0-Paket wurde bislang **nicht** auf der Test-CCU installiert.

## 6. WebUI, Browser und Zugriffsschutz — bestanden

47. Anmeldung als Admin an der Test-OpenCCU im Firefox auf Samsung-Tablet erfolgreich; CCU-Modbus-Einstellungen mit gültiger Sitzung geladen.
48. Gerätedaten geladen; Zustand „Verbunden“, acht Werte; alle GOOD.
49. Dialoge „Gerät bearbeiten“, „Register bearbeiten“, „Gerät hinzufügen“ lassen sich öffnen; Abbrechen verwirft Änderungen.
50. Im langen Geräte-Dialog sind „Abbrechen“ und „Übernehmen“ auf dem Tablet sichtbar und erreichbar.
51. Paginierung innerhalb der Registerliste über Schaltflächen „Vorherige Register“/„Weitere Register“ funktioniert: unterster Eintrag **Ausgang 4** erreichbar und sein Registerdialog geöffnet.
52. „Verbindung testen“ bei unverändertem Gerät: erfolgreich, zuvor fälschlich `connection refused`.
53. „Wert testen“ für Coil-Ausgang: erfolgreich, Rohwert `false`, Wert `false`, zuvor fälschlich `connection refused`.
54. Eigene Test-Schaltflächen erkennen Änderungen an Verbindungsparametern (Unit-ID 1 → 2) und testen dann separat; Änderung verworfen.
55. WebUI ohne gültige Sitzung: `index.cgi` und API bei GET und POST **HTTP 403**.
56. WebUI mit gültiger Sitzung: GET `status` und `config` **HTTP 200**.
57. Unbekannte API-Aktion: **HTTP 400**.
58. Mutation per falscher Methode: **HTTP 405**.
59. Beschädigtes JSON bei `save`, `test_connection`, `test_register`: **HTTP 400** mit korrekter JSON-Fehlermeldung (nach Fehlerbehebung im CGI-Proxy).
60. Zu große Anfrage (>1 MiB): **HTTP 413**.
61. Reguläres Abmelden im OpenCCU-Browser; die zuvor gültige Sitzung wird danach bei Status- und Konfigurations-GET mit **HTTP 403** abgewiesen.
62. JavaScript- und CSS-Ressourcen vom Tablet-/MSI-Netz erreichbar (**HTTP 200**).

## 7. Noch nicht bestanden oder nicht durchführbar

- **Original-CCU3** nicht im Haus; insbesondere Tcl-8.2-Laufzeittest nicht durchführbar.
- **x86_64-OpenCCU** nicht im Haus; Linux-amd64-Build erfolgreich, realer Laufzeittest nicht durchführbar.
- **0.2.0-Paketinstallation auf der Test-CCU:** Installationsversuch wurde durch Sicherheitskontrolle des Fernzugriffs blockiert. Daher keine bestätigte `0.2.0`-Installation auf der OpenCCU.
- **Container-E2E** / Installer-Replay in Docker/Podman und Playwright-Browserautomation: nicht durchgeführt bzw. nicht nachgewiesen.
- **Komplette OpenCCU-Backup-/Restore-Runde** sowie Ein-Klick-Selbstupdate: nicht durchgeführt.
- **Langzeitlast**, IPv6, alle unterstützten Datentypen und Registervarianten auf physischer Hardware, mehrere eigenständige Modbus-TCP-Clients: nicht vollständig geprüft.
- **Weitere spezielle Fehlerfälle** (z.B. beschädigte Persistenzdateien): nicht end-to-end geprüft.

## 8. Freigabestatus

Der Stand ist als **lokaler Test-Build 0.2.0** erfolgreich gebaut. Die umfangreichen Hardware- und Browserprüfungen erfolgten am inhaltlich entsprechenden vorangegangenen Test-Build mit Kennung 0.1.32. Die Änderung 0.2.0 auf dem Zielsystem, die beiden fehlenden Hardwareplattformen und die genannten erweiterten E2E-Tests sind noch offen. Es erfolgte **kein Commit, kein Push und keine öffentliche Veröffentlichung**.