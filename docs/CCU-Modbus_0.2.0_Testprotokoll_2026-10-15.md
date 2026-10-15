# CCU-Modbus 0.2.0 — Testprotokoll

**Stand:** 15. Oktober 2026 · **Release:** [v0.2.0](https://github.com/WolfHenk/CCU-Modbus/releases/tag/v0.2.0)
**Prüfgrundlage:** [homematic-community/ccu-addon-howto, Kapitel 08: Testing](https://github.com/homematic-community/ccu-addon-howto/blob/master/docs/08-testing.md); ergänzend Installation, RC-Skript, WebUI, Systemintegration und Updates.
**Testumgebung:** Browserprüfungen auf einem **Tablet unter Firefox (Android 6.0.1)**. Alle übrigen Prüfungen wurden nach Angabe des Projektverantwortlichen auf einem **Orange Pi 3B** ausgeführt; Zielsystem der Integrationsprüfungen war eine separate **OpenCCU auf ARM64**. Für Modbus-E/A wurde ein dediziertes Vierkanal-Testgerät mit vier Eingängen und vier schaltbaren Ausgängen verwendet. Private Gerätenamen, Hostnamen, IP-Adressen und lokale Dateipfade sind nicht Teil dieses öffentlichen Berichts.
**Zuordnung:** Die Punkte 47–62 umfassen Browser-/WebUI-Prüfungen (teilweise zusätzlich von der Testumgebung aus geprüft); die Punkte 1–46 wurden ohne Browser durchgeführt, soweit der einzelne Prüfpunkt nicht ausdrücklich eine Browseraktion dokumentiert (insbesondere 44–45). **Browserinteraktionen einschließlich 44–45 fanden auf dem Tablet mit Firefox statt.**
**Versionsabgrenzung:** Der Quellstand **0.2.0** wurde erfolgreich getestet und das Installationspaket einschließlich dieses Berichts erstellt und als GitHub-Release veröffentlicht. Die Integrations- und Hardwaretests auf der ARM64-OpenCCU wurden mit dem vorherigen, inhaltlich entsprechenden Test-Build (Anzeige 0.1.32) durchgeführt. **Die Installation des veröffentlichten Releasepakets 0.2.0 auf der ARM64-Test-OpenCCU wurde anschließend über die reguläre WebUI erfolgreich durchgeführt und unabhängig auf Versionskennung, Dienst und Modbus-Funktion geprüft.** Die unten genannten bestandenen Prüfungen sind daher keine Behauptung einer vollständigen plattformübergreifenden Abnahme.

**Legende:** **BESTANDEN** = tatsächlich geprüft und mit erwarteter Beobachtung abgeschlossen; **NICHT DURCHGEFÜHRT** = kein Testnachweis; **NICHT VERFÜGBAR** = benötigte Hardware fehlt. Die Punkte 1–62 sind die dokumentierten bestandenen Einzelprüfungen, nicht 62 voneinander unabhängige Tests.

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
19. Realer Vierkanal-Modbus-Baustein eingerichtet, dem Modbus-TCP-Testgerät (Port 502), Unit 1; Seriennummer **`MB003206502001`**.
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
46. Aktualisierungen mit der inhaltlich geprüften Korrektur und nachfolgender Konfigurationserhaltung erfolgreich.

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
62. JavaScript- und CSS-Ressourcen vom Tablet-Browser erreichbar (**HTTP 200**).

## 7. Installation des veröffentlichten Pakets 0.2.0 — bestanden

**63. Installation und Funktionskontrolle des GitHub-Releasepakets 0.2.0 auf ARM64-OpenCCU: BESTANDEN.** Das Paket wurde über die reguläre OpenCCU-WebUI hochgeladen und die Installation manuell gestartet. Die OpenCCU bestätigte den erfolgreichen Installationsabschluss. Nach der Installation wurden die Versionsdatei (**0.2.0**), der Dienststatus (**running**), genau ein registrierter XML-RPC-Schnittstelleneintrag, das XML-RPC-Gerät mit neun Kanälen sowie das reale Modbus-Testgerät kontrolliert. Das Gerät war **ONLINE**, alle acht Register meldeten **GOOD**, der Fehlerzähler stand auf **0**, und alle vier Ausgänge waren **AUS**.

*Abgrenzung:* Ein vollständiger **Systemneustart unmittelbar nach der endgültigen 0.2.0-Installation** wurde nicht gesondert geprüft. Der automatische Dienststart nach Neustart wurde bereits mit dem vorherigen inhaltlich entsprechenden Test-Build bestätigt.

## 8. Nicht durchgeführte oder nicht nachgewiesene Prüfungen

- **NICHT VERFÜGBAR — Original-CCU3** nicht im Haus; insbesondere Tcl-8.2-Laufzeittest nicht durchführbar.
- **NICHT VERFÜGBAR — x86_64-OpenCCU** nicht im Haus; Linux-amd64-Build erfolgreich, realer Laufzeittest nicht durchführbar.
- **NICHT DURCHGEFÜHRT — Container-E2E** / Installer-Replay in Docker/Podman und Playwright-Browserautomation: nicht durchgeführt bzw. nicht nachgewiesen.
- **NICHT DURCHGEFÜHRT — Komplette OpenCCU-Backup-/Restore-Runde** sowie Ein-Klick-Selbstupdate: nicht durchgeführt.
- **NICHT DURCHGEFÜHRT — Langzeitlast**, IPv6, alle unterstützten Datentypen und Registervarianten auf physischer Hardware, mehrere eigenständige Modbus-TCP-Clients: nicht vollständig geprüft.
- **NICHT DURCHGEFÜHRT — Weitere spezielle Fehlerfälle** (z.B. beschädigte Persistenzdateien): nicht end-to-end geprüft.
- **NICHT DURCHGEFÜHRT — Systemneustart unmittelbar nach der endgültigen 0.2.0-Releaseinstallation**; Neustarts mit dem inhaltlich entsprechenden Test-Build wurden zuvor erfolgreich geprüft.

## 9. Veröffentlichungs- und Abnahmestatus

**GitHub-Release v0.2.0 veröffentlicht.** Der zugehörige GitHub-Actions-Workflow hat das versionsgebundene Installationsarchiv und eine SHA-256-Prüfsumme bereitgestellt. Dieses Protokoll wird im Repository und im Archiv mitgeführt.

**Bestanden und im Einzelnen dokumentiert:** Punkte 1–63. Die Punkte 1–62 umfassen die bisherigen Quellcode-, Integrations-, Modbus- und Firefox-Browserprüfungen; Punkt 63 bestätigt die Installation des endgültigen Releasepakets 0.2.0 auf der ARM64-Test-OpenCCU. **Nicht durchgeführt/nicht nachgewiesen:** sämtliche in Abschnitt 8 ausdrücklich gekennzeichneten Prüfungen. Cross-Builds ersetzen keine Laufzeittests auf der originalen CCU3 oder auf einer x86_64-OpenCCU.
