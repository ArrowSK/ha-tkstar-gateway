package gt06
import"testing"
func TestCRC(t *testing.T){if got:=crc16([]byte("123456789"));got!=0x906e{t.Fatalf("%04x",got)}}
