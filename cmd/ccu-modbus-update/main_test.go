package main

import (
 "archive/tar"
 "bytes"
 "compress/gzip"
 "io"
 "strings"
 "testing"
)

func sampleArchive(version, name string) []byte {
 var b bytes.Buffer
 gz:=gzip.NewWriter(&b)
 tarw:=tar.NewWriter(gz)
 v:=[]byte(version+"\n")
 _=tarw.WriteHeader(&tar.Header{Name:"VERSION",Mode:0644,Size:int64(len(v))})
 _,_=tarw.Write(v)
 if name!="" {
  _=tarw.WriteHeader(&tar.Header{Name:name,Mode:0644,Size:1})
  _,_=io.Copy(tarw,strings.NewReader("x"))
 }
 _=tarw.Close()
 _=gz.Close()
 return b.Bytes()
}
func TestSemanticVersions(t *testing.T) {
 v,e:=newer("0.2.10","0.2.9");if e!=nil||!v{t.Fatalf("expected newer: %v",e)}
 v,e=newer("0.2.0","0.2.1");if e!=nil||v{t.Fatalf("unexpected newer: %v",e)}
 _,e=newer("0.2.1;evil","0.2.0");if e==nil{t.Fatal("injected version accepted")}
}
func TestArchiveChecks(t *testing.T) {
 for _,tc:=range []struct{n,version,extra string; good bool}{
 {"valid","0.2.1","bin/ccu-modbusd",true},
 {"wrong version","0.2.0","bin/ccu-modbusd",false},
 {"traversal","0.2.1","../escape",false},
 {"absolute","0.2.1","/tmp/escape",false},
 } {
  t.Run(tc.n,func(t *testing.T){
   e:=validateArchive(sampleArchive(tc.version,tc.extra),"0.2.1")
   if (e==nil)!=tc.good{t.Fatalf("unexpected result: %v",e)}
  })
 }
}
