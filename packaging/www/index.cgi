#!/bin/tclsh
source ../lib/session.tcl

set sid [query_value sid]
if {![check_session $sid]} {
    puts "Status: 403 Forbidden\r"
    puts "Content-Type: text/html; charset=utf-8\r"
    puts "\r"
    puts "<h2>Sitzung ungueltig</h2><p>Bitte diese Seite schliessen und CCU-Modbus erneut aus der Systemsteuerung oeffnen.</p>"
    exit 0
}

puts "Content-Type: text/html; charset=utf-8\r"
puts "Cache-Control: no-store\r"
puts "\r"
puts {<!doctype html>}
puts {<html lang="de"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">}
puts {<title>CCU-Modbus</title><link rel="stylesheet" href="style.css?v=0.1.0-test19"></head><body>}
puts {<header class="top"><div><img src="img/modbus-logo.png" class="logo" alt="Modbus"></div><div><h1>CCU-Modbus</h1><div class="sub">Modbus TCP Interface für CCU/OpenCCU</div></div><div class="version">0.1.0-test19</div></header>}
puts {<main><div id="notice" class="notice hidden"></div>}
puts {<section class="toolbar"><div><h2>Modbus-Geräte</h2><div class="hint">Geräte und Werte werden lokal auf der CCU gespeichert.</div></div><button id="addDevice" class="primary">+ Gerät hinzufügen</button></section>}
puts {<div id="devices" class="devices"><div class="empty">Lade Konfiguration…</div></div>}
puts {<section class="footerbar"><button id="saveAll" class="primary">Änderungen speichern</button><span id="dirtyText" class="muted">Keine ungespeicherten Änderungen.</span></section>}
puts {<footer class="copyright">© 2026 Wolfram Henkel · PolyForm Noncommercial 1.0.0</footer>}
puts {</main>}
puts {<dialog id="deviceDialog"><form method="dialog" id="deviceForm"><div class="dialoghead"><h2 id="deviceTitle">Gerät</h2><button value="cancel" class="iconbtn">×</button></div>}
puts {<div class="grid"><label>Name<input id="dName" required></label><label>IP-Adresse / Host<input id="dHost" required></label><label>Port<input id="dPort" type="number" min="1" max="65535" value="502"></label><label>Unit-ID<input id="dUnit" type="number" min="0" max="255" value="1"></label></div>}
puts {<details><summary>Erweitert</summary><div class="grid advanced"><label>Timeout (ms)<input id="dTimeout" type="number" min="100" max="30000" value="1500"></label><label>Retries<input id="dRetries" type="number" min="0" max="2" value="1"></label><label class="check"><input id="dEnabled" type="checkbox" checked> Gerät aktiv</label></div></details>}
puts {<div class="testline"><button type="button" id="testConnection">Verbindung testen</button><span id="connectionResult"></span></div>}
puts {<div class="registerHead"><h3>Register</h3><button type="button" id="addRegister">+ Register hinzufügen</button></div>}
puts {<div class="tablewrap"><table><thead><tr><th>Name</th><th>Typ</th><th>Adresse</th><th>Datentyp</th><th>Faktor</th><th>Intervall</th><th></th></tr></thead><tbody id="registerRows"></tbody></table></div>}
puts {<div class="dialogactions"><button type="button" id="deleteDevice" class="danger">Gerät löschen</button><span class="spacer"></span><button value="cancel">Abbrechen</button><button type="button" id="applyDevice" class="primary">Übernehmen</button></div></form></dialog>}
puts {<dialog id="registerDialog"><form method="dialog"><div class="dialoghead"><h2 id="registerTitle">Register</h2><button value="cancel" class="iconbtn">×</button></div>}
puts {<div class="grid"><label>Bezeichnung<input id="rName" required></label><label>Registertyp<select id="rType"><option value="holding">Holding Register</option><option value="input">Input Register</option><option value="coil">Coil</option><option value="discrete">Discrete Input</option></select></label><label>Registernummer<input id="rAddress" type="number" min="0" max="65535" value="0"></label><label>Datentyp<select id="rDatatype"><option>INT16</option><option>UINT16</option><option>INT32</option><option>UINT32</option><option>FLOAT32</option><option>BOOL</option></select></label><label>Einheit<input id="rUnit" placeholder="z.B. °C"></label><label>Faktor<input id="rFactor" value="1"></label><label>Offset<input id="rOffset" value="0"></label><label>Polling (s)<input id="rPoll" type="number" min="1" max="86400" value="10"></label></div>}
puts {<div class="seriesbox"><label class="check"><input id="rSeries" type="checkbox"> Mehrere aufeinanderfolgende Register anlegen</label><label id="rSeriesCountWrap" style="display:none">Gesamtanzahl<input id="rSeriesCount" type="number" min="2" max="256" value="2"></label></div><details><summary>Erweitert</summary><div class="grid advanced"><label class="check"><input id="rByteSwap" type="checkbox"> Byte-Swap</label><label class="check"><input id="rWordSwap" type="checkbox"> Word-Swap</label><label class="check"><input id="rEnabled" type="checkbox" checked> Register aktiv</label></div></details>}
puts {<div class="testline"><button type="button" id="testRegister">Wert testen</button><span id="registerResult"></span></div>}
puts {<div class="dialogactions"><button type="button" id="deleteRegister" class="danger">Register löschen</button><span class="spacer"></span><button value="cancel">Abbrechen</button><button type="button" id="applyRegister" class="primary">Übernehmen</button></div></form></dialog>}
puts "<script>window.CCU_MODBUS_SID='$sid';</script>"
puts {<script src="app.js?v=0.1.0-test19"></script></body></html>}
