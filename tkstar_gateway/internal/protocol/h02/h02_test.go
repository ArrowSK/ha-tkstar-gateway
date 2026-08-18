package h02
import"testing"
func TestV1(t *testing.T){m,e:=Parse("*HQ,1234567890,V1,101930,A,4730.0000,N,01903.0000,E,10.0,180,180826,0,0,0,0,0,80#");if e!=nil{t.Fatal(e)};if m.Position==nil||m.Position.Latitude<47.49||m.Position.Latitude>47.51{t.Fatalf("%+v",m.Position)}}
