package watch

import (
 "bufio"
 "context"
 "encoding/hex"
 "errors"
 "fmt"
 "io"
 "log/slog"
 "net"
 "strconv"
 "strings"
 "time"
 "github.com/ArrowSK/ha-tkstar-gateway/internal/model"
 "github.com/ArrowSK/ha-tkstar-gateway/internal/protocol"
)

type Server struct{ Addr string }
func (s Server) Name() string { return "watch" }
func (s Server) Run(ctx context.Context, sink protocol.Sink, log *slog.Logger) error {
 ln, err := net.Listen("tcp", s.Addr); if err != nil { return err }; defer ln.Close()
 go func(){ <-ctx.Done(); _ = ln.Close() }()
 log.Info("tracker listener started", "protocol", s.Name(), "addr", s.Addr)
 for { c, err := ln.Accept(); if err != nil { if ctx.Err()!=nil { return nil }; return err }; go s.handle(c,sink,log) }
}
func (s Server) handle(c net.Conn, sink protocol.Sink, log *slog.Logger) {
 defer c.Close(); r:=bufio.NewReaderSize(c,8192)
 for { _=c.SetReadDeadline(time.Now().Add(3*time.Minute)); raw,err:=r.ReadString(']'); if err!=nil { if !errors.Is(err,io.EOF){log.Debug("watch connection ended","error",err)}; return }
  f,err:=ParseFrame(strings.TrimSpace(raw)); if err!=nil { log.Warn("invalid watch frame","error",err); return }
  if !sink.Accept(f.DeviceID,s.Name()){log.Warn("rejected unknown tracker","protocol",s.Name());return}
  msg,ack,err:=f.Message(); if err!=nil { log.Debug("unsupported watch packet","command",f.Command(),"error",err);continue }
  if ack!="" { _=c.SetWriteDeadline(time.Now().Add(10*time.Second)); if _,err=io.WriteString(c,ack);err!=nil{return} }
  sink.Handle(msg)
 }
}

type Frame struct{ Vendor,DeviceID,LengthHex,Content string }
func ParseFrame(raw string)(Frame,error){
 if len(raw)<7||raw[0]!='['||raw[len(raw)-1]!=']'{return Frame{},errors.New("bad framing")}
 p:=strings.SplitN(raw[1:len(raw)-1],"*",4); if len(p)!=4||p[1]==""{return Frame{},errors.New("bad fields")}
 b,err:=hex.DecodeString(p[2]); if err!=nil||len(b)!=2{return Frame{},errors.New("bad length")}; n:=int(b[0])<<8|int(b[1]); if n!=len([]byte(p[3])){return Frame{},fmt.Errorf("length mismatch: %d != %d",n,len([]byte(p[3])))}
 return Frame{p[0],p[1],p[2],p[3]},nil
}
func (f Frame) Command()string{if i:=strings.IndexByte(f.Content,',');i>=0{return f.Content[:i]};return f.Content}
func (f Frame) response(cmd string)string{return fmt.Sprintf("[%s*%s*%04X*%s]",f.Vendor,f.DeviceID,len([]byte(cmd)),cmd)}
func (f Frame) Message()(model.Message,string,error){
 m:=model.Message{DeviceID:f.DeviceID,Protocol:"watch"}; cmd:=f.Command(); switch cmd {
 case "LK": fs:=strings.Split(f.Content,","); if len(fs)>1 { if b,e:=strconv.ParseFloat(fs[len(fs)-1],64);e==nil&&b>=0&&b<=100{m.Battery=&b} }; return m,f.response("LK"),nil
 case "CCID": fs:=strings.SplitN(f.Content,",",2); if len(fs)==2{m.ICCID=strings.TrimSpace(fs[1])}; return m,"",nil
 case "UD","UD2","AL": p,alarm,err:=parseLocation(f.DeviceID,f.Content); if err!=nil{return model.Message{},"",err};m.Position=p;m.Alarm=alarm;if p.Battery>=0{b:=p.Battery;m.Battery=&b};if p.GSM>=0{g:=p.GSM;m.GSM=&g};ack:="";if cmd=="AL"{ack=f.response("AL")};return m,ack,nil
 default:return model.Message{},"",fmt.Errorf("unsupported command %s",cmd)
 }
}
func parseLocation(id,content string)(*model.Position,string,error){
 f:=strings.Split(content,","); if len(f)<14{return nil,"",errors.New("short location packet")}; ts,err:=time.ParseInLocation("020106150405",f[1]+f[2],time.UTC);if err!=nil{ts=time.Now().UTC()}
 p:=&model.Position{DeviceID:id,Timestamp:ts,ReceivedAt:time.Now().UTC(),Valid:strings.EqualFold(f[3],"A"),Source:"GPS",Battery:-1,GSM:-1,Protocol:"watch"}
 p.Latitude,_=strconv.ParseFloat(f[4],64);if strings.EqualFold(f[5],"S"){p.Latitude=-p.Latitude};p.Longitude,_=strconv.ParseFloat(f[6],64);if strings.EqualFold(f[7],"W"){p.Longitude=-p.Longitude}
 if v,e:=strconv.ParseFloat(f[8],64);e==nil{p.SpeedKmh=v*1.609344};p.Heading,_=strconv.ParseFloat(f[9],64);p.Altitude,_=strconv.ParseFloat(f[10],64);if v,e:=strconv.Atoi(f[11]);e==nil{p.Satellites=v};if v,e:=strconv.Atoi(f[12]);e==nil{p.GSM=v};if v,e:=strconv.ParseFloat(f[13],64);e==nil{p.Battery=v}
 alarm:=""; if len(f)>16 { if n,e:=strconv.ParseUint(strings.TrimPrefix(f[16],"0x"),16,32);e==nil{var a []string;if n&(1<<0)!=0||n&(1<<17)!=0{a=append(a,"low_battery")};if n&(1<<16)!=0{a=append(a,"sos")};if n&(1<<18)!=0{a=append(a,"geofence_out")};if n&(1<<19)!=0{a=append(a,"geofence_in")};if n&(1<<20)!=0{a=append(a,"take_off")};alarm=strings.Join(a,",")}}
 if strings.HasPrefix(f[0],"AL")&&alarm==""{alarm="alarm"};p.Alarm=alarm
 if len(f)>22 {mcc,_:=strconv.Atoi(f[19]);mnc,_:=strconv.Atoi(f[20]);for i:=21;i+2<len(f);i+=3{lac,e1:=strconv.Atoi(f[i]);cid,e2:=strconv.Atoi(f[i+1]);sig,e3:=strconv.Atoi(f[i+2]);if e1==nil&&e2==nil&&lac>0&&cid>0{if e3!=nil{sig=0};p.Cells=append(p.Cells,model.Cell{MCC:mcc,MNC:mnc,LAC:lac,ID:cid,Signal:sig})}}}
 if !p.Valid{p.Source="LBS";p.Latitude=0;p.Longitude=0};return p,alarm,nil
}
