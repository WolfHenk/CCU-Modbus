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
    if {[catch {set result [exec /usr/bin/curl -sS --max-time 35 -H "Content-Type: application/json" -X POST --data-binary @$tmp "$base$endpoint"]} err]} {
        set result "{\"ok\":false,\"error\":\"Daemon nicht erreichbar\"}"
        set code 1
    }
    file delete -force $tmp
} else {
    if {[catch {set result [exec /usr/bin/curl -sS --max-time 5 "$base$endpoint"]} err]} {
        set result "{\"ok\":false,\"error\":\"Daemon nicht erreichbar\"}"
        set code 1
    }
}

puts "\r"
puts $result

if {$action == "save" && $code == 0 && [string first {"ok":true} $result] >= 0} {
    catch {exec /usr/local/etc/config/rc.d/ccu-modbus restart &}
}
