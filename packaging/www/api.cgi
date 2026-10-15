#!/bin/tclsh
source ../lib/session.tcl

set sid [query_value sid]
set action [query_value action]

puts "Content-Type: application/json; charset=utf-8\r"
puts "Cache-Control: no-store\r"

if {![check_session $sid]} {
    puts "Status: 403 Forbidden\r"
    puts "\r"
    puts {{"ok":false,"error":"Sitzung ungueltig"}}
    exit 0
}

set method "GET"
if {[info exists ::env(REQUEST_METHOD)]} { set method $::env(REQUEST_METHOD) }

# Update operations are authenticated separately and never accept arbitrary URLs.
if {$action == "update_status" || $action == "update_check" || $action == "update_start"} {
    if {$action == "update_start" && $method != "POST"} {
        puts "Status: 405 Method Not Allowed\r"
        puts "\r"
        puts {{"ok":false,"error":"POST erforderlich"}}
        exit 0
    }
    if {$action != "update_start" && $method != "GET"} {
        puts "Status: 405 Method Not Allowed\r"
        puts "\r"
        puts {{"ok":false,"error":"GET erforderlich"}}
        exit 0
    }
    set cmd "status"
    if {$action == "update_check"} {set cmd "check"}
    if {$action == "update_start"} {set cmd "start"}
    if {[catch {set result [exec /usr/local/addons/ccu-modbus/bin/ccu-modbus-update $cmd]} err]} {
        puts "Status: 503 Service Unavailable\r"
        puts "\r"
        puts {{"ok":false,"error":"Update-Dienst nicht erreichbar"}}
    } else {
        puts "\r"
        puts $result
    }
    exit 0
}

set base "http://127.0.0.1:18701"
set endpoint ""
set needPost 0
if {$action == "status"} {
    set endpoint "/status"
} elseif {$action == "config"} {
    set endpoint "/config"
} elseif {$action == "save"} {
    set endpoint "/config"
    set needPost 1
} elseif {$action == "test_connection"} {
    set endpoint "/test-connection"
    set needPost 1
} elseif {$action == "test_register"} {
    set endpoint "/test-register"
    set needPost 1
} else {
    puts "Status: 400 Bad Request\r"
    puts "\r"
    puts {{"ok":false,"error":"Unbekannte Aktion"}}
    exit 0
}

if {$needPost && $method != "POST"} {
    puts "Status: 405 Method Not Allowed\r"
    puts "\r"
    puts {{"ok":false,"error":"POST erforderlich"}}
    exit 0
}

set tmp ""
set code 0
if {$needPost} {
    set len 0
    if {[info exists ::env(CONTENT_LENGTH)]} { set len $::env(CONTENT_LENGTH) }
    if {$len > 1048576} {
        puts "Status: 413 Payload Too Large\r"
        puts "\r"
        puts {{"ok":false,"error":"Anfrage zu gross"}}
        exit 0
    }
    set body [read stdin $len]
    set tmp "/usr/local/tmp/ccu-modbus-[pid].json"
    set f [open $tmp w]
    puts -nonewline $f $body
    close $f
    if {[catch {set result [exec /usr/bin/env curl -sS --max-time 35 -H "Content-Type: application/json" -X POST --data-binary @$tmp -w "\n%{http_code}" "$base$endpoint"]} err]} {
        set result "{\"ok\":false,\"error\":\"Daemon nicht erreichbar\"}"
        set code 1
    }
    file delete -force $tmp
} else {
    if {[catch {set result [exec /usr/bin/env curl -sS --max-time 5 -w "\n%{http_code}" "$base$endpoint"]} err]} {
        set result "{\"ok\":false,\"error\":\"Daemon nicht erreichbar\"}"
        set code 1
    }
}

# Preserve the daemon's HTTP status. Curl exits successfully on HTTP 4xx,
# so an error JSON body alone is insufficient for clients and browser tests.
set httpStatus 502
if {!$code && [regexp {\n([0-9]{3})$} $result full statusCode]} {
    set httpStatus [expr {$statusCode + 0}]
    set result [string range $result 0 [expr {[string length $result] - [string length $full] - 1}]]
}
if {$httpStatus != 200} {
    puts "Status: $httpStatus\r"
}
puts "\r"
puts $result

if {$action == "save" && $code == 0 && [string first {"ok":true} $result] >= 0} {
    catch {exec /usr/local/etc/config/rc.d/ccu-modbus restart &}
}
