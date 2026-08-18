package watch
import("fmt";"testing")
func TestParseLocationAndCells(t *testing.T){content:="UD,180826,101930,A,47.500000,N,19.050000,E,10.0,152,100,9,80,90,0,0,00000000,2,1,216,30,1234,5678,75,1235,5679,65";raw:=fmt.Sprintf("[CS*1234567890*%04X*%s]",len(content),content);f,err:=ParseFrame(raw);if err!=nil{t.Fatal(err)};m,_,err:=f.Message();if err!=nil{t.Fatal(err)};if m.Position==nil||!m.Position.Valid{t.Fatal("expected valid position")};if len(m.Position.Cells)!=2{t.Fatalf("cells=%d",len(m.Position.Cells))}}
