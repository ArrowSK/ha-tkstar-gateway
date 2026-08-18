package main

import(
 "context"
 "flag"
 "fmt"
 "log/slog"
 "os"
 "os/signal"
 "strconv"
 "strings"
 "syscall"
 "time"
 "github.com/ArrowSK/ha-tkstar-gateway/internal/gateway"
 "github.com/ArrowSK/ha-tkstar-gateway/internal/geolocation"
 "github.com/ArrowSK/ha-tkstar-gateway/internal/mqtt"
 "github.com/ArrowSK/ha-tkstar-gateway/internal/protocol"
 "github.com/ArrowSK/ha-tkstar-gateway/internal/protocol/gt06"
 "github.com/ArrowSK/ha-tkstar-gateway/internal/protocol/h02"
 "github.com/ArrowSK/ha-tkstar-gateway/internal/protocol/watch"
 "github.com/ArrowSK/ha-tkstar-gateway/internal/store"
 "github.com/ArrowSK/ha-tkstar-gateway/internal/webui"
)
const version="0.1.0"
func main(){self:=flag.Bool("self-test",false,"run startup self-test and exit");flag.Parse();if *self{fmt.Println("TKSTAR Gateway self-test OK",version);return};level:=slog.LevelInfo;switch strings.ToLower(env("TKSTAR_LOG_LEVEL","info")){case"debug":level=slog.LevelDebug;case"warn":level=slog.LevelWarn;case"error":level=slog.LevelError};log:=slog.New(slog.NewTextHandler(os.Stdout,&slog.HandlerOptions{Level:level}));data:=env("TKSTAR_DATA_DIR","/data");st,err:=store.New(data,envInt("TKSTAR_HISTORY_DAYS",180));fatal(log,"store",err);_=st.Cleanup(time.Now());geo:=geolocation.New(data,os.Getenv("TKSTAR_GOOGLE_GEOLOCATION_API_KEY"));gw:=gateway.New(st,geo,log,time.Duration(envInt("TKSTAR_OFFLINE_AFTER_MINUTES",10))*time.Minute);mc:=mqtt.New(mqtt.Config{Host:env("TKSTAR_MQTT_HOST","core-mosquitto"),Port:envInt("TKSTAR_MQTT_PORT",1883),Username:os.Getenv("TKSTAR_MQTT_USERNAME"),Password:os.Getenv("TKSTAR_MQTT_PASSWORD")},log,gw.SetPrivacyByKey);if err:=mc.Start();err!=nil{log.Warn("MQTT not ready; receiver continues","error",err)}else{gw.SetMQTT(mc)};ctx,cancel:=signal.NotifyContext(context.Background(),syscall.SIGINT,syscall.SIGTERM);defer cancel();servers:=[]protocol.Server{watch.Server{Addr:env("TKSTAR_WATCH_ADDR",":5093")},h02.Server{Addr:env("TKSTAR_H02_ADDR",":5013")},gt06.Server{Addr:env("TKSTAR_GT06_ADDR",":5023")}};for _,srv:=range servers{s:=srv;go func(){if e:=s.Run(ctx,gw,log);e!=nil&&ctx.Err()==nil{log.Error("tracker listener stopped","protocol",s.Name(),"error",e);cancel()}}()};go func(){if e:=webui.Listen(env("TKSTAR_HTTP_ADDR",":8099"),webui.New(gw,log).Handler(),log);e!=nil&&ctx.Err()==nil{log.Error("web server stopped","error",e);cancel()}}();tick:=time.NewTicker(time.Minute);cleanup:=time.NewTicker(6*time.Hour);defer tick.Stop();defer cleanup.Stop();for{select{case<-ctx.Done():log.Info("stopping");return;case<-tick.C:gw.MarkOffline();case<-cleanup.C:_=st.Cleanup(time.Now())}}}
func env(k,d string)string{if v:=os.Getenv(k);v!=""{return v};return d};func envInt(k string,d int)int{v,e:=strconv.Atoi(os.Getenv(k));if e==nil&&v>0{return v};return d};func fatal(l *slog.Logger,w string,e error){if e!=nil{l.Error(w+" failed","error",e);os.Exit(1)}}
