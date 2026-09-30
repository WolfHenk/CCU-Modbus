# CCU-Modbus Architekturregeln

1. Kein externer Dienst ist fuer den Betrieb erforderlich.
2. Kein CCU/RPC-Aufruf darf auf eine Modbus-Antwort warten.
3. Ein Modbus-Geraet darf kein anderes Geraet blockieren.
4. Ein fehlerhaftes Register darf keinen Daemon-Absturz verursachen.
5. Gueltige Geraete laufen trotz fehlerhafter Konfiguration anderer Geraete weiter.
6. Online-Geraeteprofile werden nur als Importquelle genutzt; Laufzeit ist immer lokal.
7. Schreibzugriffe sind standardmaessig deaktiviert.
8. Normale Logs enthalten keine Frame-Dumps.
9. Persistente Daten liegen unter /usr/local/etc/config/addons/ccu-modbus.
10. Die WebUI ist Konfiguration/Diagnose, nicht Teil des Echtzeitpfades.
